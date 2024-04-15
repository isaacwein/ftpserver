// Description: This is the main file of the ftp server
// The main function starts the ftp server and the ftps server
// The ftp server is started on port 21 and the ftps server is started on port 990
// The ftp server is started with the TryListenAndServeTLSe function and the ftps server is started with the TryListenAndServeTLS function
// The ftp server and the ftpes server are started with the TryListenAndServe function
// The ftpes server are started with the TryListenAndServeTLSe function
// The ftps server are started with the TryListenAndServeTLS function
// you can run it with the docker-compose file in the root of the project

package main

import (
	"context"
	"fmt"
	"github.com/lmittmann/tint"
	"github.com/telebroad/fileserver/filesystem"
	"github.com/telebroad/fileserver/ftp"
	"github.com/telebroad/fileserver/httphandler"
	"github.com/telebroad/fileserver/sftp"
	"github.com/telebroad/fileserver/users"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func ExampleServer() {

	// setting up the slog logger
	logger := setupLogger()
	slog.SetDefault(logger)

	logger.Debug("Starting FTP server")
	env, err := GetEnv(logger)
	if err != nil {
		logger.Error("Error getting environment", "error", err)
		os.Exit(1)
	}

	// create a new user
	u := GetUsers(logger)

	// file system
	fs := filesystem.NewLocalFS(env.FtpServerRoot)

	// ftp server
	ftpServer, err := ftp.NewServer(env.FtpAddr, fs, u)
	if err != nil {
		fmt.Println("Error starting ftp server", "error", err)
		return
	}
	ftpServer.SetLogger(logger.With("module", "ftp-server"))
	err = ftpServer.SetPublicServerIPv4(env.FtpServerIPv4)
	if err != nil {
		fmt.Println("Error setting public server ip", "error", err)
		return
	}
	// setting the passive ports range
	ftpServer.PasvMinPort = env.PasvMinPort
	ftpServer.PasvMaxPort = env.PasvMaxPort

	err = ftpServer.TryListenAndServeTLSe(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting ftp server", "error", err)
		return
	}

	logger.Info("FTP server started", "port", env.FtpAddr)

	ftpsServer, err := ftp.NewServer(env.FtpsAddr, fs, u)
	err = ftpServer.SetPublicServerIPv4(env.FtpServerIPv4)
	if err != nil {
		logger.Error("Error setting public server ip", "error", err)
		return
	}
	ftpsServer.SetLogger(logger.With("module", "ftps-server"))
	ftpsServer.PasvMinPort = env.PasvMinPort
	ftpsServer.PasvMaxPort = env.PasvMaxPort
	err = ftpsServer.TryListenAndServeTLS(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting ftps server", "error", err)
		return
	}

	logger.Info("FTPS server started", "port", env.FtpsAddr)

	// sftp server

	sftpServer := sftp.NewSFTPServer(env.SftpAddr, fs, u)

	sftpServer.SetLogger(logger.With("module", "sftp-server"))
	err = sftpServer.SetPrivateKeyFile(env.KeyFile)
	if err != nil {
		logger.Error("Error setting private key", "error", err)
		return
	}
	err = sftpServer.TryListenAndServe(time.Second)
	if err != nil {
		logger.Error("Error starting sftp server", "error", err)
		return
	}
	logger.Info("SFTP server started", "port", env.SftpAddr)
	router := http.NewServeMux()

	router.Handle("/static/{$}", httphandler.NewFileServerHandler("/static/", fs, nil))
	router.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Welcome to the filesystem server")
	})

	httpServer := &httphandler.Server{
		Server: &http.Server{
			Addr:    os.Getenv("HTTP_SERVER_ADDR"),
			Handler: router,
		},
	}

	err = httpServer.TryListenAndServe(time.Second)
	if err != nil {
		logger.Error("Error starting http server", "error", err)
	}

	httpsServer := &httphandler.Server{
		Server: &http.Server{
			Addr:    os.Getenv("HTTPS_SERVER_ADDR"),
			Handler: router,
		}}

	err = httpsServer.TryListenAndServeTLS(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting https server", "error", err)
	}

	// graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
	ftpServer.Close(fmt.Errorf("ftp server closed by signal"))
	ftpsServer.Close(fmt.Errorf("ftps server closed by signal"))
	sftpServer.Close()
	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("http server closed by signal"))
	defer cancel()
	httpServer.Shutdown(ctx)
	httpsServer.Shutdown(ctx)
}

func setupLogger() *slog.Logger {
	logLevel := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	}

	handlerOptions := &tint.Options{
		AddSource:   true,
		Level:       logLevel, // Only log messages of level INFO and above
		ReplaceAttr: nil,
	}

	handler := tint.NewHandler(os.Stdout, handlerOptions)

	logger := slog.New(handler).With("app", "filesystem-server")
	logger.Handler()
	logger.Info("Logger initialized", "level", logLevel)

	return logger
}

// GetUsers returns a new ftp.Users with the default user
func GetUsers(logger *slog.Logger) ftp.Users {
	Users := users.NewLocalUsers(logger)
	// load the default user
	FtpDefaultUser := os.Getenv("FTP_DEFAULT_USER")
	FtpDefaultPass := os.Getenv("FTP_DEFAULT_PASS")
	FtpDefaultIp := os.Getenv("FTP_DEFAULT_IP")
	logger.Debug("FTP_DEFAULT_USER is", "username", FtpDefaultUser)
	logger.Debug("FTP_DEFAULT_PASS is", "password", FtpDefaultPass)
	logger.Debug("FTP_DEFAULT_IP is", "Allowed form origin IP", FtpDefaultIp)
	user1 := Users.Add(FtpDefaultUser, FtpDefaultPass)
	user1.AddIP("127.0.0.0/8")
	user1.AddIP("10.0.0.0/8")
	user1.AddIP("172.16.0.0/12")
	user1.AddIP("192.168.0.0/16")
	user1.AddIP("fd00::/8")
	user1.AddIP("::1")

	return Users
}

// Environment is the environment of the server
type Environment struct {
	FtpAddr       string
	FtpsAddr      string
	SftpAddr      string
	CrtFile       string
	KeyFile       string
	FtpServerIPv4 string
	FtpServerRoot string
	PasvMinPort   int
	PasvMaxPort   int
}

// GetEnv returns a new Environment with the environment variables
func GetEnv(logger *slog.Logger) (env *Environment, err error) {
	env = &Environment{}
	// this is the bublic ip of the server FOR PASV mode
	env.FtpServerIPv4 = os.Getenv("FTP_SERVER_IPV4")
	if env.FtpServerIPv4 == "" {

		// Set a default FTP_SERVER_IPV4 if the environment variable is not set
		fmt.Println("FTP_SERVER_IPV4 was empty so Getting public ip from ipify.org...")
		env.FtpServerIPv4, err = ftp.GetServerPublicIP()
		if err != nil {
			return nil, fmt.Errorf("error getting public ip: %w", err)
		}
		// Set a default port if the environment variable is not set
	}
	env.FtpAddr = os.Getenv("FTP_SERVER_ADDR")
	env.FtpsAddr = os.Getenv("FTPS_SERVER_ADDR")
	env.SftpAddr = os.Getenv("SFTP_SERVER_ADDR")
	env.FtpServerRoot = os.Getenv("FTP_SERVER_ROOT")

	logger.Debug("FTP_SERVER_ADDR is", "ADDR", env.FtpAddr)
	logger.Debug("FTPS_SERVER_ADDR is", "ADDR", env.FtpsAddr)
	logger.Debug("FTP_SERVER_IPV4 is", "IP", env.FtpServerIPv4)
	logger.Debug("FTP_SERVER_ROOT is", "ROOT", env.FtpServerRoot)

	// convert port string to int
	env.PasvMinPort, _ = strconv.Atoi(os.Getenv("PASV_MIN_PORT"))

	env.PasvMaxPort, _ = strconv.Atoi(os.Getenv("PASV_MAX_PORT"))

	logger.Debug("PASV_MIN_PORT is", "PORT", env.PasvMinPort)
	logger.Debug("PASV_MAX_PORT is", "PORT", env.PasvMaxPort)

	// load the crt and key files
	env.CrtFile = os.Getenv("CRT_FILE")
	logger.Debug("CRT_FILE is ", env.CrtFile)
	env.KeyFile = os.Getenv("KEY_FILE")
	logger.Debug("KEY_FILE is ", env.KeyFile)

	return
}

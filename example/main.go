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
	"embed"
	"fmt"
	"github.com/lmittmann/tint"
	"github.com/telebroad/fileserver/filesystem"
	"github.com/telebroad/fileserver/ftp"
	"github.com/telebroad/fileserver/httphandler"
	"github.com/telebroad/fileserver/sftp"
	"github.com/telebroad/fileserver/users"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var (
	//go:embed keys
	keysDir embed.FS
)

func main() {

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
	localFS := filesystem.NewLocalFS(env.FtpServerRoot)

	// ftp server
	ftpServer, err := ftp.NewServer(env.FtpAddr, localFS, u)
	if err != nil {
		fmt.Println("Error starting ftp server", "error", err)
		return
	}
	ftpServer.SetLogger(logger.With("module", "ftp-server"))
	// seting the public server ip for passive mode
	err = ftpServer.SetPublicServerIPv4(env.FtpServerIPv4)
	if err != nil {
		fmt.Println("Error setting public server ip", "error", err)
		return
	}
	// setting the passive ports range
	ftpServer.PasvMinPort = env.PasvMinPort
	ftpServer.PasvMaxPort = env.PasvMaxPort

	// starting the ftp server
	// TLSe means that the server start with tls encryption but it can be upgraded to tls encryption
	err = ftpServer.TryListenAndServeTLSe(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting ftp server", "error", err)
		return
	}
	logger.Info("FTP server started", "port", env.FtpAddr)

	// ftps server
	ftpsServer, err := ftp.NewServer(env.FtpsAddr, localFS, u)
	err = ftpServer.SetPublicServerIPv4(env.FtpServerIPv4)
	if err != nil {
		logger.Error("Error setting public server ip", "error", err)
		return
	}
	ftpsServer.SetLogger(logger.With("module", "ftps-server"))
	ftpsServer.PasvMinPort = env.PasvMinPort
	ftpsServer.PasvMaxPort = env.PasvMaxPort
	// ONLY ACCEPT TLS CONNECTIONS
	err = ftpsServer.TryListenAndServeTLS(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting ftps server", "error", err)
		return
	}

	logger.Info("FTPS server started", "port", env.FtpsAddr)

	// sftp server
	sftpServer := sftp.NewSFTPServer(env.SftpAddr, localFS, u)

	sftpServer.SetLogger(logger.With("module", "sftp-server"))
	// adding a directory with private keys
	// ecdsa, rsa, ed25519
	fs.WalkDir(keysDir, ".", func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}
		file, err := fs.ReadFile(keysDir, path)
		if err != nil {
			return err
		}
		sftpServer.SetPrivateKey(path, file)
		return nil
	})
	// starting the sftp server
	err = sftpServer.TryListenAndServe(time.Second)
	if err != nil {
		logger.Error("Error starting sftp server", "error", err)
		return
	}
	logger.Info("SFTP server started", "port", env.SftpAddr)

	// http file server support read and write and delete files
	router := http.NewServeMux()

	router.Handle("/static/{pathname...}", httphandler.NewFileServerHandler("/static", localFS, u))
	httpServer := &httphandler.Server{
		Server: &http.Server{
			Addr:    os.Getenv("HTTP_SERVER_ADDR"),
			Handler: router,
		},
	}
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><body>")
		fmt.Fprintf(w, "<h1>Welcome to the filesystem server</h1>")
		fmt.Fprintf(w, `<h2>file server is at <a href="/static">/static</a></h2>`)
		fmt.Fprintf(w, "</body></html>")
	})
	// try is the same of listen and serve but with a timeout if no error is returned it returns nil
	err = httpServer.TryListenAndServe(time.Second)
	if err != nil {
		logger.Error("Error starting http server", "error", err)
	}
	// https server
	httpsServer := &httphandler.Server{
		Server: &http.Server{
			Addr:    os.Getenv("HTTPS_SERVER_ADDR"),
			Handler: router,
		}}

	err = httpsServer.TryListenAndServeTLS(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting https server", "error", err)
	}

	// graceful shutdown all servers
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
	AddSource := false
	switch os.Getenv("LOG_LEVEL") {

	case "DEBUG":
		logLevel = slog.LevelDebug
		AddSource = true
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	}

	handlerOptions := &tint.Options{
		AddSource:   AddSource,
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
func GetUsers(logger *slog.Logger) *users.LocalUsers {
	Users := users.NewLocalUsers(logger)
	// load the default user
	DefaultUser := os.Getenv("DEFAULT_USER")
	DefaultPass := os.Getenv("DEFAULT_PASS")
	DefaultIp := os.Getenv("DEFAULT_IP")
	DefaultIps := strings.Split(DefaultIp, ",")
	logger.Debug("DEFAULT_USER is", "username", DefaultUser)
	logger.Debug("DEFAULT_PASS is", "password", DefaultPass)
	logger.Debug("DEFAULT_IP is", "Allowed form origin IPs", DefaultIp)
	if DefaultUser == "" || DefaultPass == "" {
		logger.Info("DEFAULT_USER or DEFAULT_PASS is empty, not creating default user")
		return Users
	}
	user1 := Users.Add(DefaultUser, DefaultPass)

	for _, ip := range DefaultIps {
		if ip == "" {
			continue
		}
		user1.AddIP(strings.Trim(ip, " \n\r\t"))
	}

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
	logger.Debug("CRT_FILE is ", "file", env.CrtFile)
	env.KeyFile = os.Getenv("KEY_FILE")
	logger.Debug("KEY_FILE is ", "file", env.KeyFile)

	return
}

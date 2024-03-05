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
	"fmt"
	"github.com/telebroad/ftpserver/ftp"
	"github.com/telebroad/ftpserver/ftp/ftpusers"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func main() {
	// setting up the slog logger
	handlerOptions := &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelDebug, // Only log messages of level INFO and above
		ReplaceAttr: nil,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOptions)).With("app", "ftp-server")
	slog.SetDefault(logger)

	logger.Debug("Starting FTP server")
	env, err := GetEnv(logger)
	if err != nil {
		logger.Error("Error getting environment", "error", err)
		os.Exit(1)
	}

	// create a new user
	users := GetUsers(logger)

	ftpServer, err := ftp.NewServer(env.FtpAddr, ftp.NewFtpLocalFS(env.FtpServerRoot), users)
	if err != nil {
		fmt.Println("Error starting ftp server", "error", err)
		return
	}
	ftpServer.SetLogger(logger)
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

	ftpsServer, err := ftp.NewServer(env.FtpsAddr, ftp.NewFtpLocalFS(env.FtpServerRoot), users)
	err = ftpServer.SetPublicServerIPv4(env.FtpServerIPv4)
	if err != nil {
		logger.Error("Error setting public server ip", "error", err)
		return
	}
	ftpsServer.SetLogger(logger)
	ftpsServer.PasvMinPort = env.PasvMinPort
	ftpsServer.PasvMaxPort = env.PasvMaxPort
	err = ftpsServer.TryListenAndServeTLS(env.CrtFile, env.KeyFile, time.Second)
	if err != nil {
		logger.Error("Error starting ftps server", "error", err)
		return
	}

	logger.Info("FTPS server started", "port", env.FtpsAddr)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
	ftpServer.Close(fmt.Errorf("server closed by signal"))

}

// GetUsers returns a new ftp.Users with the default user
func GetUsers(logger *slog.Logger) ftp.Users {
	Users := ftpusers.NewLocalUsers()
	// load the default user
	FtpDefaultUser := os.Getenv("FTP_DEFAULT_USER")
	FtpDefaultPass := os.Getenv("FTP_DEFAULT_PASS")
	FtpDefaultIp := os.Getenv("FTP_DEFAULT_IP")

	if FtpDefaultUser != "" {
		FtpDefaultUser = "user"
	}

	if FtpDefaultPass != "" {
		FtpDefaultPass = "password"
	}

	if FtpDefaultIp != "" {
		FtpDefaultIp = "127.0.0.0/8"
	}
	logger.Info("FTP_DEFAULT_USER is", "username", FtpDefaultUser)
	logger.Info("FTP_DEFAULT_PASS is", "password", FtpDefaultPass)
	logger.Info("FTP_DEFAULT_IP is", "Allowed form origin IP", FtpDefaultIp)
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
	if env.FtpAddr == "" {
		// Set a default port if the environment variable is not set
		env.FtpAddr = ":21"
	}
	env.FtpsAddr = os.Getenv("FTPS_SERVER_ADDR")
	if env.FtpsAddr == "" {
		// Set a default port if the environment variable is not set
		env.FtpsAddr = ":990"
	}
	env.SftpAddr = os.Getenv("SFTP_SERVER_ADDR")
	if env.SftpAddr == "" {
		// Set a default port if the environment variable is not set
		env.SftpAddr = ":22"
	}
	env.FtpServerRoot = os.Getenv("FTP_SERVER_ROOT")
	if env.FtpServerRoot == "" {
		// Set a default port if the environment variable is not set
		env.FtpServerRoot = "/static"
	}

	logger.Info("FTP_SERVER_ADDR is", "ADDR", env.FtpAddr)
	logger.Info("FTPS_SERVER_ADDR is", "ADDR", env.FtpsAddr)
	logger.Info("FTP_SERVER_IPV4 is", "IP", env.FtpServerIPv4)
	logger.Info("FTP_SERVER_ROOT is", "ROOT", env.FtpServerRoot)

	pasvMinPort := os.Getenv("PASV_MIN_PORT")
	if pasvMinPort == "" {
		// Set a default port if the environment variable is not set
		fmt.Println("PASV_MIN_PORT default to 30000")
		pasvMinPort = "30000"
	}
	pasvMaxPort := os.Getenv("PASV_MAX_PORT")
	if pasvMaxPort == "" {
		fmt.Println("PASV_MAX_PORT default to 30009")
		// Set a default port if the environment variable is not set
		pasvMaxPort = "30009"
	}

	// convert port string to int
	env.PasvMinPort, err = strconv.Atoi(pasvMinPort)
	if err != nil {
		return nil, fmt.Errorf("error converting PASV_MIN_PORT to int: %w", err)
	}

	env.PasvMaxPort, err = strconv.Atoi(pasvMaxPort)
	if err != nil {
		return nil, fmt.Errorf("error converting PASV_MAX_PORT to int: %w", err)
	}
	logger.Info("PASV_MIN_PORT is", "PORT", env.PasvMinPort)
	logger.Info("PASV_MAX_PORT is", "PORT", env.PasvMaxPort)
	// load the crt and key files
	env.CrtFile = os.Getenv("CRT_FILE")
	logger.Info("CRT_FILE is ", env.CrtFile)
	env.KeyFile = os.Getenv("KEY_FILE")
	logger.Info("KEY_FILE is ", env.KeyFile)

	return
}

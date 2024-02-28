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
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func main() {
	handlerOptions := &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelDebug, // Only log messages of level INFO and above
		ReplaceAttr: nil,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOptions)).With("app", "ftp-server")
	slog.SetDefault(logger)

	logger.Debug("Starting FTP server")
	env, err := GetEnv()
	if err != nil {
		logger.Error("Error getting environment", "error", err)
		os.Exit(1)
	}
	// create a new user

	users := GetUsers()

	ftpServer, err := ftp.NewServer(env.FtpPort, ftp.NewFtpLocalFS(env.FtpServerRoot), users)
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

	logger.Info("FTP server started", "port", env.FtpPort)

	ftpsServer, err := ftp.NewServer(env.FtpsPort, ftp.NewFtpLocalFS(env.FtpServerRoot), users)
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

	logger.Info("FTPS server started", "port", env.FtpsPort)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
	ftpServer.Close(fmt.Errorf("server closed by signal"))

}

func GetUsers() ftp.Users {
	Users := ftpusers.NewLocalUsers()
	user1 := Users.Add("user", "password", 1)
	user1.AddIP("127.0.0.0/8")
	user1.AddIP("10.0.0.0/8")
	user1.AddIP("172.16.0.0/12")
	user1.AddIP("192.168.0.0/16")
	user1.AddIP("fd00::/8")
	user1.AddIP("::1")
	return Users
}

type Environment struct {
	FtpPort       string
	FtpsPort      string
	CrtFile       string
	KeyFile       string
	FtpServerIPv4 string
	FtpServerRoot string
	PasvMinPort   int
	PasvMaxPort   int
}

func GetEnv() (env *Environment, err error) {
	env = &Environment{}
	// this is the bublic ip of the server FOR PASV mode
	env.FtpServerIPv4 = os.Getenv("FTP_SERVER_IPV4")
	if env.FtpServerIPv4 == "" {

		// Set a default FTP_SERVER_IPV4 if the environment variable is not set
		fmt.Println("FTP_SERVER_IPV4 was empty so Getting public ip from ipify.org...")
		ipifyRes, err := http.Get("https://api.ipify.org")
		if err != nil {
			return nil, fmt.Errorf("error getting public ip: %w", err)
		}
		ftpServerIPv4b, err := io.ReadAll(ipifyRes.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading public ip: %w", err)
		}
		env.FtpServerIPv4 = string(ftpServerIPv4b)
		fmt.Println("FTP_SERVER_IPV4 is ", env.FtpServerIPv4)
		// Set a default port if the environment variable is not set
	}
	env.FtpPort = os.Getenv("FTP_SERVER_PORT")
	if env.FtpPort == "" {
		// Set a default port if the environment variable is not set
		env.FtpPort = ":21"
		fmt.Println("FTP_SERVER_PORT default to :21")
	}
	env.FtpsPort = os.Getenv("FTPS_SERVER_PORT")
	if env.FtpsPort == "" {
		// Set a default port if the environment variable is not set
		env.FtpsPort = ":990"
		fmt.Println("FTPS_SERVER_PORT default to :990")
	}
	env.FtpServerRoot = os.Getenv("FTP_SERVER_ROOT")
	if env.FtpServerRoot == "" {
		// Set a default port if the environment variable is not set
		env.FtpServerRoot = "/static"
		fmt.Println("FTP_SERVER_ROOT default to /static")
	}

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
	// convert to int
	env.PasvMinPort, err = strconv.Atoi(pasvMinPort)
	if err != nil {
		return nil, fmt.Errorf("error converting PASV_MIN_PORT to int: %w", err)
	}
	env.PasvMaxPort, err = strconv.Atoi(pasvMaxPort)
	if err != nil {

		return nil, fmt.Errorf("error converting PASV_MAX_PORT to int: %w", err)
	}

	env.CrtFile = "tls/ssl-rsa/localhost.rsa.crt"
	env.KeyFile = "tls/ssl-rsa/localhost.rsa.key"
	return
}

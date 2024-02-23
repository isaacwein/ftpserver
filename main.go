package main

import (
	"fmt"
	"github.com/telebroad/ftpserver/server"
	"github.com/telebroad/ftpserver/users"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func main() {

	// this is the bublic ip of the server FOR PASV mode
	ftpServerIPv4 := os.Getenv("FTP_SERVER_IPV4")
	if ftpServerIPv4 == "" {

		// Set a default FTP_SERVER_IPV4 if the environment variable is not set
		fmt.Println("FTP_SERVER_IPV4 was empty so Getting public ip from ipify.org...")
		ipifyRes, err := http.Get("https://api.ipify.org")
		if err != nil {
			fmt.Println("Error getting public ip", "error", err)
			return
		}
		ftpServerIPv4b, err := io.ReadAll(ipifyRes.Body)
		if err != nil {
			fmt.Println("Error reading public ip", "error", err)
			return
		}
		ftpServerIPv4 = string(ftpServerIPv4b)
		fmt.Println("FTP_SERVER_IPV4 is ", ftpServerIPv4)
		// Set a default port if the environment variable is not set
	}
	ftpPort := os.Getenv("FTP_SERVER_PORT")
	if ftpPort == "" {
		// Set a default port if the environment variable is not set
		ftpPort = ":21"
	}
	ftpServerRoot := os.Getenv("FTP_SERVER_ROOT")
	if ftpServerRoot == "" {
		// Set a default port if the environment variable is not set
		ftpServerRoot = "/static"
	}

	pasvMinPort := os.Getenv("PASV_MIN_PORT")
	if pasvMinPort == "" {
		// Set a default port if the environment variable is not set
		fmt.Println("PASV_MIN_PORT was empty so setting it to 30000")
		pasvMinPort = "30000"
	}
	pasvMaxPort := os.Getenv("PASV_MAX_PORT")
	if pasvMaxPort == "" {
		fmt.Println("PASV_MAX_PORT was empty so setting it to 30009")
		// Set a default port if the environment variable is not set
		pasvMaxPort = "30009"
	}
	// convert to int
	pasvMinPortP, err := strconv.Atoi(pasvMinPort)
	if err != nil {
		fmt.Println("Error converting PASV_MIN_PORT  to int", "error", err)
		return
	}
	pasvMaxPortP, err := strconv.Atoi(pasvMaxPort)
	if err != nil {
		fmt.Println("Error converting PASV_MAX_PORT to int", "error", err)
		return
	}

	// create a new user
	Users := users.NewLocalUsers()
	user1 := Users.Add("user", "password", 1)
	user1.AddIP("127.0.0.1")
	user1.AddIP("::1")

	ftpServer, err := server.NewServer(ftpPort, server.NewFtpLocalFS(ftpServerRoot, "/"), Users)
	if err != nil {
		fmt.Println("Error starting ftp server", "error", err)
		return
	}
	err = ftpServer.SetPublicServerIPv4(ftpServerIPv4)
	if err != nil {
		fmt.Println("Error setting public server ip", "error", err)
		return
	}

	ftpServer.PasvMinPort = pasvMinPortP
	ftpServer.PasvMaxPort = pasvMaxPortP

	err = ftpServer.TryListenAndServe(time.Second)
	if err != nil {
		return
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
	ftpServer.Close(fmt.Errorf("server closed by signal"))

}

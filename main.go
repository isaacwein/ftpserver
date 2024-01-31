package main

import (
	"github.com/telebroad/ftpserver/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
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

	ftpServer := server.NewFTPServer("21", server.NewFtpLocalFS(ftpServerRoot))
	ftpServer.Start()
	stopChan := make(chan os.Signal, 1)
	signal.Notify(
		stopChan,
		syscall.SIGHUP,  // (0x1) Terminal hangup
		syscall.SIGINT,  // (0x2) Interrupt from keyboard (Ctrl+C)
		syscall.SIGQUIT, // (0x3) Quit from keyboard
		syscall.SIGABRT, // (0x6) Aborted (core dumped)
		syscall.SIGKILL, // (0x9) Killed (cannot be caught)
		syscall.SIGTERM, // (0xf) Terminated (generic termination signal)
	)

	<-stopChan
	ftpServer.Stop()
}

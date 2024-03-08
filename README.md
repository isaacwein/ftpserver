# fileserver
## a go implementation of a simple FTP FTPES FTPS SFTP file server, also a handler for http file upload


### to run the basic app with docker-compose
it will run the app and a ftp client to test the server
```bash
docker compose up
```

documentation for ftp and ftps directory [ftp ](ftp/README.md)
documentation for sftp is in directory [sftp](sftp/README.md)


example setting basic user [users](users/README.md)

```go
package main

import (
	"github.com/telebroad/ftpserver/ftp"
	"github.com/telebroad/ftpserver/users"
	"log/slog"
	"os"
)

// GetUsers returns a new ftp.Users with the default user
func GetUsers(logger *slog.Logger) ftp.Users {
	Users := users.NewLocalUsers()
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

```

ftp server with a basic user
```go

package main
```
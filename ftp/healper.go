package ftp

import (
	"fmt"
	"io"
	"net/http"
)

// PublicIpUrl is the url to get the public ip of the server
const PublicIpUrl = "https://api.ipify.org"

// GetServerPublicIP returns the public IP of the server

func GetServerPublicIP() (string, error) {
	ipifyRes, err := http.Get(PublicIpUrl)
	if err != nil {
		return "", fmt.Errorf("error getting public ip: %w", err)
	}
	ftpServerIPv4, err := io.ReadAll(ipifyRes.Body)
	if err != nil {
		return "", fmt.Errorf("error reading public ip: %w", err)
	}
	return string(ftpServerIPv4), nil
}

package sshclient

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Timeout time.Duration
	Port    int
}

func LoadConfig() Config {
	timeout := 5 * time.Second
	if v := os.Getenv("SSH_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}

	port := 22
	if v := os.Getenv("SSH_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			port = n
		}
	}

	return Config{
		Timeout: timeout,
		Port:    port,
	}
}

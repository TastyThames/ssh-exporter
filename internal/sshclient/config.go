package sshclient

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Timeout time.Duration
	Port    int

	// lab-friendly switch; prod ควร false แล้วใช้ known_hosts จริง
	InsecureSkipHostKey bool
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

	insecure := false
	if v := os.Getenv("SSH_INSECURE_SKIP_HOSTKEY"); v != "" {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "1" || v == "true" || v == "yes" {
			insecure = true
		}
	}

	return Config{
		Timeout:             timeout,
		Port:                port,
		InsecureSkipHostKey: insecure,
	}
}

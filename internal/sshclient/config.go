package sshclient

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	User           string
	KeyPath        string
	Port           int
	Timeout        time.Duration
	KnownHostsPath string
}

func LoadConfig() Config {
	return Config{
		User:           getenv("SSH_USER", "ubuntu"),
		KeyPath:        getenv("SSH_KEY_PATH", os.ExpandEnv("$HOME/.ssh/id_ed25519")),
		Port:           getenvInt("SSH_PORT", 22),
		Timeout:        getenvDuration("SSH_TIMEOUT", 5*time.Second),
		KnownHostsPath: getenv("SSH_KNOWN_HOSTS", os.ExpandEnv("$HOME/.ssh/known_hosts")),
	}
}

func getenv(k, fb string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fb
}

func getenvInt(k string, fb int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fb
}

func getenvDuration(k string, fb time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fb
}

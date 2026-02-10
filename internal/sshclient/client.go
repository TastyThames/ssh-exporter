package sshclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	cfg Config
}

func New(cfg Config) (*Client, error) {
	return &Client{cfg: cfg}, nil
}

func (c *Client) RunPassword(ctx context.Context, host, user, password, cmd string) (string, error) {
	if user == "" {
		user = "root"
	}
	if password == "" {
		return "", fmt.Errorf("empty password")
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", c.cfg.Port))

	sshCfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // B-mode (internal use)
		Timeout:         c.cfg.Timeout,
	}

	dialer := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	cc, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		return "", err
	}
	client := ssh.NewClient(cc, chans, reqs)
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}

// helper: ใช้ timeout ปลอดภัย
func WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

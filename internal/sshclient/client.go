package sshclient

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Client struct {
	cfg    Config
	signer ssh.Signer
	hk     ssh.HostKeyCallback
}

func New(cfg Config) (*Client, error) {
	keyBytes, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", cfg.KeyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	hk, err := knownhosts.New(cfg.KnownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("knownhosts %q: %w", cfg.KnownHostsPath, err)
	}

	return &Client{cfg: cfg, signer: signer, hk: hk}, nil
}

func (c *Client) dial(ctx context.Context, host string) (*ssh.Client, error) {
	addr := net.JoinHostPort(host, fmt.Sprint(c.cfg.Port))

	sshCfg := &ssh.ClientConfig{
		User:            c.cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(c.signer)},
		HostKeyCallback: c.hk,
		Timeout:         c.cfg.Timeout, // coarse timeout
	}

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	cc, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh handshake %s: %w", addr, err)
	}
	return ssh.NewClient(cc, chans, reqs), nil
}

func (c *Client) Run(ctx context.Context, host string, cmd AllowedCommand) (string, error) {
	cli, err := c.dial(ctx, host)
	if err != nil {
		return "", err
	}
	defer cli.Close()

	sess, err := cli.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	type out struct {
		s   string
		err error
	}
	ch := make(chan out, 1)

	go func() {
		b, err := sess.CombinedOutput(cmd.String())
		ch <- out{s: string(b), err: err}
	}()

	select {
	case <-ctx.Done():
		_ = cli.Close()
		return "", ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return r.s, fmt.Errorf("run %q: %w; output=%q", cmd.String(), r.err, truncate(r.s, 400))
		}
		return r.s, nil
	case <-time.After(c.cfg.Timeout):
		_ = cli.Close()
		return "", fmt.Errorf("timeout after %s", c.cfg.Timeout)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

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

// RunPassword executes cmd on host using username/password (per-target).
func (c *Client) RunPassword(ctx context.Context, host, user, password, cmd string) (string, error) {
	if user == "" {
		return "", fmt.Errorf("ssh user is empty")
	}
	if password == "" {
		return "", fmt.Errorf("ssh password is empty")
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", c.cfg.Port))

	// HostKey policy (lab-first)
	// TODO: harden later with known_hosts
	hk := ssh.InsecureIgnoreHostKey()
	_ = c.cfg.InsecureSkipHostKey // (kept for future: enforce known_hosts when false)

	sshCfg := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: hk,
		Timeout:         c.cfg.Timeout,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(_user, _instruction string, questions []string, _echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			}),
		},
	}

	// Dial with context so it won't hang forever.
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Make sure the underlying TCP conn obeys ctx timeout too.
	// (ssh handshake can still hang without deadlines)
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.cfg.Timeout))
	}

	cconn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		return "", err
	}
	client := ssh.NewClient(cconn, chans, reqs)
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	type result struct {
		out []byte
		err error
	}
	done := make(chan result, 1)

	go func() {
		out, err := sess.CombinedOutput(cmd)
		done <- result{out: out, err: err}
	}()

	select {
	case <-ctx.Done():
		// Best-effort terminate session.
		_ = sess.Signal(ssh.SIGKILL)
		return "", ctx.Err()
	case r := <-done:
		if r.err != nil {
			// return output for debugging context too
			return string(r.out), r.err
		}
		return string(r.out), nil
	}
}

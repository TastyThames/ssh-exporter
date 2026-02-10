package scheduler

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tastythames/ssh-exporter/internal/cache"
	"github.com/tastythames/ssh-exporter/internal/sshclient"
)

func StartWorker(id int, jobs <-chan Job, c cache.Cache) {
	log.Printf("worker %d started", id)

	cfg := sshclient.LoadConfig()
	cli, err := sshclient.New(cfg)
	if err != nil {
		log.Printf("worker %d: ssh client init error: %v", id, err)
		return
	}

	for job := range jobs {
		host := strings.TrimSpace(job.Target)

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		start := time.Now()

		res := cache.Result{
			At:     time.Now(),
			Labels: job.Labels,
			Values: map[string]float64{
				"ssh_target_scrape_duration_seconds":       0,
				"ssh_target_last_scrape_timestamp_seconds": float64(time.Now().Unix()),
				"ssh_target_up": 0,
			},
		}

		// only support password_env ตอนนี้
		if job.AuthMode != "" && job.AuthMode != "password_env" {
			res.Err = &ErrString{S: "unsupported auth mode: " + job.AuthMode}
			res.Values["ssh_target_scrape_duration_seconds"] = time.Since(start).Seconds()
			cancel()
			c.Set(job.Target, res)
			continue
		}

		if job.PasswordEnv == "" {
			res.Err = &ErrString{S: "missing password_env in inventory"}
			res.Values["ssh_target_scrape_duration_seconds"] = time.Since(start).Seconds()
			cancel()
			c.Set(job.Target, res)
			continue
		}

		password := os.Getenv(job.PasswordEnv)
		if password == "" {
			res.Err = &ErrString{S: "empty env var: " + job.PasswordEnv}
			res.Values["ssh_target_scrape_duration_seconds"] = time.Since(start).Seconds()
			cancel()
			c.Set(job.Target, res)
			continue
		}

		// command: uptime seconds
		out, e := cli.RunPassword(ctx, host, job.SSHUser, password, "cat /proc/uptime")
		cancel()

		res.Values["ssh_target_scrape_duration_seconds"] = time.Since(start).Seconds()

		if e != nil {
			res.Err = e
			res.Values["ssh_target_up"] = 0
			c.Set(job.Target, res)
			continue
		}

		// parse uptime: "<seconds> <idle>\n"
		// minimal parse (ปล่อย robust ทีหลัง)
		secs := parseFirstFloat(out)
		res.Values["ssh_target_up"] = 1
		res.Values["ssh_os_uptime_seconds"] = secs

		c.Set(job.Target, res)
	}
}

// ---- tiny helpers ----

type ErrString struct{ S string }

func (e *ErrString) Error() string { return e.S }

func parseFirstFloat(s string) float64 {
	// very small parser: split by space, parse first token
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return v
}

package scheduler

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tastythames/ssh-exporter/internal/cache"
	"github.com/tastythames/ssh-exporter/internal/metrics"
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
		runOneJob(id, cfg, cli, job, c)
	}
}

func runOneJob(id int, cfg sshclient.Config, cli *sshclient.Client, job Job, c cache.Cache) {
	host := strings.TrimSpace(job.Target)
	start := time.Now()

	res := cache.Result{
		At:     time.Now(),
		Labels: job.Labels,
		Values: map[string]float64{
			metrics.MetricScrapeDuration: 0,
			metrics.MetricLastScrapeTs:   0,
			metrics.MetricTargetUp:       0,
		},
	}

	// defaults
	authMode := strings.TrimSpace(job.AuthMode)
	if authMode == "" {
		authMode = "password_env"
	}
	user := strings.TrimSpace(job.SSHUser)
	if user == "" {
		user = "root" // lab default
	}

	// pick password
	password, perr := resolvePassword(authMode, job)
	if perr != nil {
		res.Err = perr
		finalizeResult(&res, start)
		c.Set(job.Target, res)
		return
	}

	// ctx per job (cancel immediately at end of this function)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	log.Printf("worker %d got job: target=%s labels=%v auth=%s", id, job.Target, job.Labels, authMode)

	out, e := cli.RunPassword(ctx, host, user, password, "cat /proc/uptime")
	finalizeResult(&res, start)

	if e != nil {
		res.Err = e
		res.Values[metrics.MetricTargetUp] = 0
		c.Set(job.Target, res)
		return
	}

	secs := parseFirstFloat(out)
	res.Values[metrics.MetricTargetUp] = 1
	res.Values["ssh_os_uptime_seconds"] = secs

	c.Set(job.Target, res)
}

func resolvePassword(authMode string, job Job) (string, error) {
	switch authMode {
	case "password_env":
		env := strings.TrimSpace(job.PasswordEnv)
		if env == "" {
			return "", &ErrString{S: "missing ssh.auth.password_env (mode=password_env)"}
		}
		pw := strings.TrimSpace(os.Getenv(env))
		if pw == "" {
			return "", &ErrString{S: "empty env var: " + env}
		}
		return pw, nil

	case "password_file":
		p := strings.TrimSpace(job.PasswordFile)
		if p == "" {
			return "", &ErrString{S: "missing ssh.auth.password_file (mode=password_file)"}
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		pw := strings.TrimSpace(string(b))
		if pw == "" {
			return "", &ErrString{S: "empty password_file: " + p}
		}
		return pw, nil

	default:
		return "", &ErrString{S: "unsupported auth mode: " + authMode}
	}
}

func finalizeResult(res *cache.Result, start time.Time) {
	res.Values[metrics.MetricScrapeDuration] = time.Since(start).Seconds()
	res.Values[metrics.MetricLastScrapeTs] = float64(time.Now().Unix())
}

// ---- tiny helpers ----

type ErrString struct{ S string }

func (e *ErrString) Error() string { return e.S }

func parseFirstFloat(s string) float64 {
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

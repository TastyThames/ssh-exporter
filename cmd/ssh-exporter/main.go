package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tastythames/ssh-exporter/internal/cache"
	"github.com/tastythames/ssh-exporter/internal/inventory"
	"github.com/tastythames/ssh-exporter/internal/metrics"
	"github.com/tastythames/ssh-exporter/internal/scheduler"
)

func getenv(k, fb string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fb
}

func main() {
	listen := getenv("EXPORTER_LISTEN", ":9222")

	invPath := getenv("INVENTORY_FILE", "deploy/targets.example.yaml")

	log.Printf("config: listen=%s inventory=%s", listen, invPath)

	inv, err := inventory.Load(invPath)
	if err != nil {
		log.Fatalf("load inventory: %v", err)
	}

	jobs := make([]scheduler.Job, 0, len(inv.Targets))
	for _, t := range inv.Targets {
		jobs = append(jobs, scheduler.Job{
			Target: t.Address,
			Labels: t.Labels,

			SSHUser: t.SSH.User,

			AuthMode:     t.SSH.Auth.Mode,
			PasswordEnv:  t.SSH.Auth.PasswordEnv,
			PasswordFile: t.SSH.Auth.PasswordFile,

			KeyPath: t.SSH.Auth.KeyPath,
		})
	}

	// 2) cache + scheduler
	c := cache.NewMemCache()

	jobCh := make(chan scheduler.Job, 100) // buffer สำคัญมาก
	sched := scheduler.NewScheduler(scheduler.Options{
		Interval: 10 * time.Second,
		Jitter:   2 * time.Second,
		JobCh:    jobCh,
	})

	// worker pool size
	workers := 5
	for i := 0; i < workers; i++ {
		go scheduler.StartWorker(i, jobCh, c)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sched.Run(ctx, jobs)

	// 3) HTTP
	r := metrics.NewRenderer(c)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		r.Write(w)
	})

	srv := &http.Server{
		Addr:    listen,
		Handler: mux,
	}

	go func() {
		log.Printf("ssh-exporter listening on %s\n", listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	log.Println("shutdown...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

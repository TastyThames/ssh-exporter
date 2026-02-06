package scheduler

import (
	"context"
	"log"
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
		log.Printf("worker %d got job: target=%s labels=%v", id, job.Target, job.Labels)

		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel() // NOTE: จะถูกเรียกเมื่อจบ iteration นี้ (เพราะเราจะ `continue` บ่อย) -> ใช้แบบ explicit ดีกว่า
		// ⚠️ ใน Go ถ้าใช้ defer ใน loop จะสะสม; เราจะใช้ cancel แบบ explicit ด้านล่างแทน

		host := strings.TrimSpace(job.Target)

		// สร้าง result โครงไว้ก่อน (duration ค่อย set ทีหลัง)
		result := cache.Result{
			At:     time.Now(),
			Labels: job.Labels,
			Values: map[string]float64{
				metrics.MetricTargetUp:     0, // default fail
				metrics.MetricLastScrapeTs: float64(time.Now().Unix()),
			},
		}

		// (แก้ defer สะสม) ใช้ cancel แบบ explicit
		cancelNow := func() {
			cancel()
		}

		// 1) uptime (ถือเป็น "primary check" ถ้าพังถือว่า target down)
		out, e := cli.Run(ctx, host, sshclient.CmdUptime())
		if e != nil {
			result.Err = e
			result.Values[metrics.MetricTargetUp] = 0
			result.Values[metrics.MetricScrapeDuration] = time.Since(start).Seconds()
			cancelNow()
			c.Set(job.Target, result)
			continue
		}

		up, e := sshclient.ParseUptimeSeconds(out)
		if e != nil {
			result.Err = e
			result.Values[metrics.MetricTargetUp] = 0
			result.Values[metrics.MetricScrapeDuration] = time.Since(start).Seconds()
			cancelNow()
			c.Set(job.Target, result)
			continue
		}

		// uptime ผ่าน = target up
		result.Values[metrics.MetricTargetUp] = 1
		result.Values["ssh_os_uptime_seconds"] = up

		// 2) load1 (optional)
		out, e = cli.Run(ctx, host, sshclient.CmdLoadavg())
		if e == nil {
			if l1, pe := sshclient.ParseLoad1(out); pe == nil {
				result.Values["ssh_os_load1"] = l1
			}
		}

		// 3) meminfo (optional)
		out, e = cli.Run(ctx, host, sshclient.CmdMeminfo())
		if e == nil {
			if total, avail, pe := sshclient.ParseMeminfo(out); pe == nil {
				result.Values["ssh_os_mem_total_bytes"] = total
				result.Values["ssh_os_mem_available_bytes"] = avail
			}
		}

		// duration สุดท้าย
		result.Values[metrics.MetricScrapeDuration] = time.Since(start).Seconds()

		cancelNow()
		c.Set(job.Target, result)
	}
}

package scheduler

import (
	"context"
	"log"
	"math/rand"
	"sync/atomic"
	"time"
)

type Scheduler struct {
	interval time.Duration
	jitter   time.Duration

	jobCh chan Job

	// stats (atomic) for observability
	enqueued uint64
	dropped  uint64
}

type Options struct {
	Interval time.Duration
	Jitter   time.Duration
	JobCh    chan Job
}

// NewScheduler creates a scheduler that periodically enqueues jobs into jobCh.
// - Interval: base schedule interval
// - Jitter: random delay added each cycle (0..Jitter) to reduce herd effects
func NewScheduler(opts Options) *Scheduler {
	if opts.Interval <= 0 {
		opts.Interval = 10 * time.Second
	}
	return &Scheduler{
		interval: opts.Interval,
		jitter:   opts.Jitter,
		jobCh:    opts.JobCh,
	}
}

func (s *Scheduler) Run(ctx context.Context, jobs []Job) {
	// Kick once immediately
	s.enqueueAll(ctx, jobs)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// add jitter (optional)
			if s.jitter > 0 {
				delay := time.Duration(rand.Int63n(int64(s.jitter)))
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			s.enqueueAll(ctx, jobs)
		}
	}
}

// enqueueAll pushes jobs into jobCh.
// IMPORTANT: This is non-blocking; if jobCh is full, we drop and count it.
// This avoids scheduler deadlock when workers are slow.
func (s *Scheduler) enqueueAll(ctx context.Context, jobs []Job) {
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case s.jobCh <- j:
			atomic.AddUint64(&s.enqueued, 1)
		default:
			atomic.AddUint64(&s.dropped, 1)
		}
	}

	// (optional) log when drops happen
	d := atomic.LoadUint64(&s.dropped)
	if d > 0 {
		log.Printf("scheduler: dropped=%d (job queue full)\n", d)
	}
}

func (s *Scheduler) Stats() (enqueued uint64, dropped uint64) {
	return atomic.LoadUint64(&s.enqueued), atomic.LoadUint64(&s.dropped)
}

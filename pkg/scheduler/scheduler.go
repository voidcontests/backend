package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/voidcontests/api/internal/lib/logger/sl"
)

type Task func(ctx context.Context) error

type Scheduler struct {
	interval time.Duration
	task     Task
	stop     chan struct{}
	done     chan struct{}
}

func New(interval time.Duration, task Task) *Scheduler {
	return &Scheduler{
		interval: interval,
		task:     task,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	slog.Info("scheduler: started", slog.String("interval", s.interval.String()))

	if err := s.task(ctx); err != nil {
		slog.Error("scheduler: task execution failed", sl.Err(err))
	}

	for {
		select {
		case <-ticker.C:
			if err := s.task(ctx); err != nil {
				slog.Error("scheduler: task execution failed", sl.Err(err))
			}
		case <-s.stop:
			slog.Info("scheduler: stopping...")
			return
		case <-ctx.Done():
			slog.Info("scheduler: context cancelled, stopping...")
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.done
	slog.Info("scheduler: stopped")
}

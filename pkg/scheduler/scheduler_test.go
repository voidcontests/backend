package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("creates scheduler with correct interval and task", func(t *testing.T) {
		interval := 100 * time.Millisecond
		task := func(ctx context.Context) error {
			return nil
		}

		s := New(interval, task)

		assert.NotNil(t, s)
		assert.Equal(t, interval, s.interval)
		assert.NotNil(t, s.task)
		assert.NotNil(t, s.stop)
		assert.NotNil(t, s.done)
	})
}

func TestScheduler_Start(t *testing.T) {
	t.Run("executes task immediately on start", func(t *testing.T) {
		executed := false
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			executed = true
			mu.Unlock()
			return nil
		}

		s := New(1*time.Hour, task)
		ctx := context.Background()

		go s.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		s.Stop()

		mu.Lock()
		assert.True(t, executed)
		mu.Unlock()
	})

	t.Run("executes task periodically", func(t *testing.T) {
		var count int
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		}

		interval := 50 * time.Millisecond
		s := New(interval, task)
		ctx := context.Background()

		go s.Start(ctx)
		time.Sleep(160 * time.Millisecond)
		s.Stop()

		mu.Lock()
		assert.GreaterOrEqual(t, count, 3)
		mu.Unlock()
	})

	t.Run("stops when Stop is called", func(t *testing.T) {
		var count int
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		}

		interval := 30 * time.Millisecond
		s := New(interval, task)
		ctx := context.Background()

		go s.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		s.Stop()

		mu.Lock()
		countAfterStop := count
		mu.Unlock()

		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, countAfterStop, count, "task should not execute after Stop")
		mu.Unlock()
	})

	t.Run("stops when context is cancelled", func(t *testing.T) {
		var count int
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		}

		interval := 30 * time.Millisecond
		s := New(interval, task)
		ctx, cancel := context.WithCancel(context.Background())

		go s.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		cancel()

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		countAfterCancel := count
		mu.Unlock()

		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, countAfterCancel, count, "task should not execute after context cancellation")
		mu.Unlock()
	})

	t.Run("continues execution when task returns error", func(t *testing.T) {
		var count int
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			count++
			mu.Unlock()
			return errors.New("test error")
		}

		interval := 50 * time.Millisecond
		s := New(interval, task)
		ctx := context.Background()

		go s.Start(ctx)
		time.Sleep(160 * time.Millisecond)
		s.Stop()

		mu.Lock()
		assert.GreaterOrEqual(t, count, 3, "scheduler should continue despite task errors")
		mu.Unlock()
	})

	t.Run("task receives correct context", func(t *testing.T) {
		type contextKey string
		key := contextKey("test-key")
		expectedValue := "test-value"

		var receivedValue string
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			if val := ctx.Value(key); val != nil {
				receivedValue = val.(string)
			}
			mu.Unlock()
			return nil
		}

		interval := 1 * time.Hour
		s := New(interval, task)
		ctx := context.WithValue(context.Background(), key, expectedValue)

		go s.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		s.Stop()

		mu.Lock()
		assert.Equal(t, expectedValue, receivedValue)
		mu.Unlock()
	})
}

func TestScheduler_Stop(t *testing.T) {
	t.Run("waits for scheduler to finish", func(t *testing.T) {
		taskStarted := make(chan struct{})
		taskCanFinish := make(chan struct{})

		task := func(ctx context.Context) error {
			close(taskStarted)
			<-taskCanFinish
			return nil
		}

		s := New(1*time.Hour, task)
		ctx := context.Background()

		go s.Start(ctx)

		<-taskStarted

		stopFinished := make(chan struct{})
		go func() {
			s.Stop()
			close(stopFinished)
		}()

		select {
		case <-stopFinished:
			t.Fatal("Stop should wait for task to finish")
		case <-time.After(50 * time.Millisecond):
		}

		close(taskCanFinish)

		select {
		case <-stopFinished:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Stop did not complete in time")
		}
	})

	t.Run("Stop waits for graceful shutdown", func(t *testing.T) {
		task := func(ctx context.Context) error {
			return nil
		}

		s := New(1*time.Hour, task)
		ctx := context.Background()

		go s.Start(ctx)
		time.Sleep(50 * time.Millisecond)

		done := make(chan struct{})
		go func() {
			s.Stop()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Stop deadlocked or hung")
		}
	})
}

func TestScheduler_Integration(t *testing.T) {
	t.Run("realistic usage scenario", func(t *testing.T) {
		var executionTimes []time.Time
		var mu sync.Mutex

		task := func(ctx context.Context) error {
			mu.Lock()
			executionTimes = append(executionTimes, time.Now())
			mu.Unlock()
			return nil
		}

		interval := 40 * time.Millisecond
		s := New(interval, task)
		ctx := context.Background()

		start := time.Now()
		go s.Start(ctx)
		time.Sleep(150 * time.Millisecond)
		s.Stop()
		elapsed := time.Since(start)

		mu.Lock()
		count := len(executionTimes)
		mu.Unlock()

		assert.GreaterOrEqual(t, count, 3)
		assert.Less(t, elapsed, 200*time.Millisecond)

		mu.Lock()
		if len(executionTimes) >= 2 {
			for i := 1; i < len(executionTimes); i++ {
				gap := executionTimes[i].Sub(executionTimes[i-1])
				assert.Greater(t, gap, 30*time.Millisecond)
				assert.Less(t, gap, 60*time.Millisecond)
			}
		}
		mu.Unlock()
	})
}

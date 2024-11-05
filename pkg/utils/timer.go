package utils

import (
	"time"

	"go.uber.org/zap"
)

type Timer struct {
	duration time.Duration
	callback func()
	timer    *time.Timer
	logger   *zap.Logger
}

func NewTimer(duration time.Duration, callback func(), logger *zap.Logger) *Timer {
	return &Timer{
		duration: duration,
		callback: callback,
		logger:   logger,
	}
}

func (t *Timer) Start() {
	t.timer = time.NewTimer(t.duration)
	go func() {
		<-t.timer.C
		t.logger.Info("Timer expired", zap.Duration("duration", t.duration))
		t.callback()
	}()
}

func (t *Timer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
		t.logger.Info("Timer stopped", zap.Duration("duration", t.duration))
	}
}

func (t *Timer) Reset() {
	if t.timer != nil {
		t.timer.Reset(t.duration)
		t.logger.Info("Timer reset", zap.Duration("duration", t.duration))
	}
}

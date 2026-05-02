package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Job struct {
	ID        string
	Type      string
	Status    Status
	Payload   []byte
	Error     string
	Attempts  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Handler func(ctx context.Context, job Job) error

type Runner struct {
	handlers map[string]Handler
	clock    func() time.Time
}

func NewRunner() *Runner {
	return &Runner{handlers: map[string]Handler{}, clock: time.Now}
}

func (r *Runner) Register(jobType string, handler Handler) error {
	if jobType == "" {
		return errors.New("job type is required")
	}
	if handler == nil {
		return errors.New("job handler is required")
	}
	r.handlers[jobType] = handler
	return nil
}

func (r *Runner) Run(ctx context.Context, job Job) (Job, error) {
	handler, ok := r.handlers[job.Type]
	if !ok {
		job.Status = StatusFailed
		job.Error = fmt.Sprintf("no handler registered for job type %q", job.Type)
		job.UpdatedAt = r.clock()
		return job, errors.New(job.Error)
	}
	job.Status = StatusRunning
	job.Attempts++
	job.UpdatedAt = r.clock()
	if err := handler(ctx, job); err != nil {
		job.Status = StatusFailed
		job.Error = err.Error()
		job.UpdatedAt = r.clock()
		return job, err
	}
	job.Status = StatusSucceeded
	job.Error = ""
	job.UpdatedAt = r.clock()
	return job, nil
}

func IsTerminal(status Status) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

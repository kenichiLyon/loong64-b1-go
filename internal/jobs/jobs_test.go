package jobs

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerMarksSuccessfulJob(t *testing.T) {
	runner := NewRunner()
	if err := runner.Register("parse", func(context.Context, Job) error { return nil }); err != nil {
		t.Fatal(err)
	}
	job, err := runner.Run(context.Background(), Job{ID: "1", Type: "parse", Status: StatusQueued})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if job.Status != StatusSucceeded || job.Attempts != 1 {
		t.Fatalf("unexpected job: %#v", job)
	}
}

func TestRunnerMarksFailedJob(t *testing.T) {
	runner := NewRunner()
	expected := errors.New("boom")
	if err := runner.Register("parse", func(context.Context, Job) error { return expected }); err != nil {
		t.Fatal(err)
	}
	job, err := runner.Run(context.Background(), Job{ID: "1", Type: "parse", Status: StatusQueued})
	if !errors.Is(err, expected) {
		t.Fatalf("expected boom, got %v", err)
	}
	if job.Status != StatusFailed || job.Error != "boom" {
		t.Fatalf("unexpected job: %#v", job)
	}
}

func TestIsTerminal(t *testing.T) {
	for _, status := range []Status{StatusSucceeded, StatusFailed, StatusCancelled} {
		if !IsTerminal(status) {
			t.Fatalf("%s should be terminal", status)
		}
	}
	if IsTerminal(StatusRunning) {
		t.Fatal("running should not be terminal")
	}
}

package health

import (
	"context"
	"errors"
	"testing"
)

type fakeChecker struct {
	name string
	err  error
}

func (f fakeChecker) Name() string                { return f.name }
func (f fakeChecker) Check(context.Context) error { return f.err }

func TestReadyReturnsFailWhenAnyCheckFails(t *testing.T) {
	service := New(fakeChecker{name: "ok"}, fakeChecker{name: "db", err: errors.New("down")})
	snapshot := service.Ready(context.Background())
	if snapshot.Status != StatusFail {
		t.Fatalf("expected fail, got %s", snapshot.Status)
	}
	if len(snapshot.Checks) != 2 || snapshot.Checks[1].Error != "down" {
		t.Fatalf("unexpected checks: %#v", snapshot.Checks)
	}
}

func TestLiveDoesNotRunChecks(t *testing.T) {
	service := New(fakeChecker{name: "db", err: errors.New("down")})
	snapshot := service.Live()
	if snapshot.Status != StatusOK {
		t.Fatalf("expected ok, got %s", snapshot.Status)
	}
	if len(snapshot.Checks) != 0 {
		t.Fatalf("live should not expose dependency checks")
	}
}

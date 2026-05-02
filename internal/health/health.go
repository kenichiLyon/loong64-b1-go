package health

import (
	"context"
	"time"
)

const (
	StatusOK   = "ok"
	StatusFail = "fail"
)

type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

type CheckResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration"`
}

type Snapshot struct {
	Status string        `json:"status"`
	Time   string        `json:"time"`
	Checks []CheckResult `json:"checks,omitempty"`
}

type Service struct {
	checks []Checker
	clock  func() time.Time
}

func New(checks ...Checker) *Service {
	return &Service{checks: checks, clock: time.Now}
}

func (s *Service) Live() Snapshot {
	return Snapshot{Status: StatusOK, Time: s.clock().UTC().Format(time.RFC3339)}
}

func (s *Service) Ready(ctx context.Context) Snapshot {
	results := make([]CheckResult, 0, len(s.checks))
	status := StatusOK
	for _, check := range s.checks {
		started := time.Now()
		err := check.Check(ctx)
		result := CheckResult{Name: check.Name(), Status: StatusOK, Duration: time.Since(started).String()}
		if err != nil {
			result.Status = StatusFail
			result.Error = err.Error()
			status = StatusFail
		}
		results = append(results, result)
	}
	return Snapshot{Status: status, Time: s.clock().UTC().Format(time.RFC3339), Checks: results}
}

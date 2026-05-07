package teaching

import (
	"context"
	"testing"
)

func TestGetBootstrapStatus(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepo{userCount: 0})
	status, err := service.GetBootstrapStatus(context.Background())
	if err != nil {
		t.Fatalf("bootstrap status: %v", err)
	}
	if status.Initialized || status.UserCount != 0 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestBootstrapCreateAdminRejectsInitializedSystem(t *testing.T) {
	t.Parallel()
	service := NewService(&fakeRepo{userCount: 1})
	_, err := service.BootstrapCreateAdmin(context.Background(), BootstrapCreateAdminInput{
		Username:    "admin1",
		DisplayName: "Admin One",
		EmployeeNo:  "A001",
	}, AuditEntry{})
	if ErrorKindOf(err) != KindConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

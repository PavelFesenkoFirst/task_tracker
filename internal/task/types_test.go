package task

import "testing"

func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		status Status
		valid  bool
	}{
		{status: StatusNew, valid: true},
		{status: StatusInProgress, valid: true},
		{status: StatusDone, valid: true},
		{status: Status("other"), valid: false},
		{status: Status(""), valid: false},
	}

	for _, tc := range tests {
		if got := tc.status.IsValid(); got != tc.valid {
			t.Fatalf("status %q: expected %v, got %v", tc.status, tc.valid, got)
		}
	}
}

func TestValidationErrorError(t *testing.T) {
	err := ValidationError{
		Field:   "status",
		Message: "must be valid",
	}

	if got := err.Error(); got != "invalid status: must be valid" {
		t.Fatalf("unexpected error string: %q", got)
	}
}

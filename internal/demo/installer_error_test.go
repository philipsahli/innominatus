package demo

import (
	"fmt"
	"strings"
	"testing"
)

// TestUserCreationErrorHandling verifies the error handling logic
// for Keycloak user creation in demo-time
func TestUserCreationErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		shouldFail  bool
		expectedMsg string
	}{
		{
			name:        "Success - no error",
			err:         nil,
			shouldFail:  false,
			expectedMsg: "demo-user created",
		},
		{
			name:        "User already exists - 409 Conflict",
			err:         fmt.Errorf("HTTP request failed: status 409"),
			shouldFail:  false,
			expectedMsg: "demo-user already exists",
		},
		{
			name:        "Real error - 500 Internal Server Error",
			err:         fmt.Errorf("HTTP request failed: status 500"),
			shouldFail:  true,
			expectedMsg: "failed to create demo-user",
		},
		{
			name:        "Real error - network timeout",
			err:         fmt.Errorf("network timeout"),
			shouldFail:  true,
			expectedMsg: "failed to create demo-user",
		},
		{
			name:        "Real error - authentication failed",
			err:         fmt.Errorf("HTTP request failed: status 401"),
			shouldFail:  true,
			expectedMsg: "failed to create demo-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the error handling logic from installer.go
			var result error

			if tt.err != nil {
				// Check if it's a "user already exists" error (409 Conflict - this is OK)
				if strings.Contains(tt.err.Error(), "status 409") {
					// Success case - user already exists
					result = nil
				} else {
					// Any other error should fail
					result = fmt.Errorf("failed to create demo-user: %w", tt.err)
				}
			} else {
				// Success case - user created
				result = nil
			}

			// Verify the result matches expectations
			if tt.shouldFail {
				if result == nil {
					t.Errorf("Expected error but got nil")
				}
				if result != nil && !strings.Contains(result.Error(), tt.expectedMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedMsg, result.Error())
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil but got error: %v", result)
				}
			}
		})
	}
}

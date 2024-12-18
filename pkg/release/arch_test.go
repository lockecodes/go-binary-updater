package release

import (
	"testing"
)

func TestMapArch(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AMD Architecture Mapping",
			input:    "amd64",
			expected: "x86_64",
		},
		{
			name:     "ARM Architecture Mapping",
			input:    "arm64",
			expected: "arm64",
		},
		{
			name:     "Unknown Architecture Mapping",
			input:    "unknown",
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := MapArch(tc.input)
			if output != tc.expected {
				t.Errorf("expected %s, but got %s", tc.expected, output)
			}
		})
	}
}

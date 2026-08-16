package system

import "testing"

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		uptime   uint64
		expected string
	}{
		{
			name:     "zero uptime",
			uptime:   0,
			expected: "0D, 0H, 0M, 0S",
		},
		{
			name:     "one second uptime",
			uptime:   1,
			expected: "0D, 0H, 0M, 1S",
		},
		{
			name:     "one minute uptime",
			uptime:   60,
			expected: "0D, 0H, 1M, 0S",
		},
		{
			name:     "one hour uptime",
			uptime:   3600,
			expected: "0D, 1H, 0M, 0S",
		},
		{
			name:     "one day uptime",
			uptime:   86400,
			expected: "1D, 0H, 0M, 0S",
		},
		{
			name:     "combined uptime",
			uptime:   90061,
			expected: "1D, 1H, 1M, 1S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUptime(tt.uptime)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

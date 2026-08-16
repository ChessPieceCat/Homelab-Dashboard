package docker

import "testing"

func TestCalculateCPUUsagePercent(t *testing.T) {
	tests := []struct {
		name     string
		previous cpuStats
		current  cpuStats
		expected float64
	}{
		{
			name:     "normal case",
			previous: makeCPUStats(100, 200),
			current:  makeCPUStats(200, 400),
			expected: 50.0,
		},
		{
			name:     "zero cpu delta",
			previous: makeCPUStats(100, 200),
			current:  makeCPUStats(100, 400),
			expected: 0.0,
		},
		{
			name:     "zero system cpu delta",
			previous: makeCPUStats(100, 200),
			current:  makeCPUStats(200, 200),
			expected: 0.0,
		},
		{
			name:     "CPU counter decreased",
			previous: makeCPUStats(200, 400),
			current:  makeCPUStats(100, 400),
			expected: 0.0,
		},
		{
			name:     "cpu usage decreased",
			previous: makeCPUStats(200, 400),
			current:  makeCPUStats(100, 500),
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCPUUsagePercent(tt.previous, tt.current)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}

}

// TestCalculateMemoryUsagePercent
func TestCalculateMemoryUsagePercent(t *testing.T) {
	tests := []struct {
		name     string
		stats    containerStats
		expected float64
	}{
		{
			name:     "normal case",
			stats:    makeMemoryStats(100, 200, map[string]uint64{"cache": 50}),
			expected: 25.0,
		},
		{
			name:     "zero used memory",
			stats:    makeMemoryStats(50, 200, map[string]uint64{"cache": 50}),
			expected: 0.0,
		},
		{
			name:     "no cache or inactive_file",
			stats:    makeMemoryStats(100, 200, map[string]uint64{}),
			expected: 50.0,
		},
		{
			name:     "cache greater than usage",
			stats:    makeMemoryStats(50, 200, map[string]uint64{"cache": 100}),
			expected: 0.0,
		},
		{
			name:     "inactive_file greater than usage",
			stats:    makeMemoryStats(50, 200, map[string]uint64{"inactive_file": 100}),
			expected: 0.0,
		},
		{
			name:     "zero memory limit",
			stats:    makeMemoryStats(100, 0, map[string]uint64{"cache": 50}),
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMemoryUsagePercent(tt.stats)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helpers for constructing containerStats test data
func makeCPUStats(totalCPU, systemCPU uint64) cpuStats {
	var stats cpuStats
	stats.CPUUsage.TotalUsage = totalCPU
	stats.SystemCPUUsage = systemCPU
	return stats
}

func makeMemoryStats(
	usage, limit uint64,
	memoryStats map[string]uint64,
) containerStats {
	var stats containerStats
	stats.MemoryStats.Usage = usage
	stats.MemoryStats.Limit = limit
	stats.MemoryStats.Stats = memoryStats
	return stats
}

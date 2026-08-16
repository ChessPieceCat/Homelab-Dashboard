package docker

import "testing"

func TestMonitorGetStatusesReturnsCopy(t *testing.T) {
	monitor := &Monitor{
		statuses: []Container{
			{
				ID:          "123",
				Name:        "test-container",
				Status:      "running",
				CPUUsage:    10,
				MemoryUsage: 20,
			},
		},
	}

	statuses := monitor.GetStatuses()

	if len(statuses) != 1 {
		t.Fatalf("expected 1 container, got %d", len(statuses))
	}

	statuses[0].Status = "modified"

	original := monitor.GetStatuses()

	if original[0].Status != "running" {
		t.Errorf(
			"modifying returned statuses changed monitor state: got %q",
			original[0].Status,
		)
	}
}

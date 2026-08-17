package main

import (
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"

	"homelab-dashboard/internal/docker"
)

func TestTemplateRendering(t *testing.T) {
	// Construct test data.
	data := DashboardData{
		Statuses: []docker.Container{
			{
				ID:          "test-container-id",
				Name:        "test-container",
				Status:      "Up",
				CPUUsage:    10.5,
				MemoryUsage: 20.25,
			},
			{
				ID:          "stopped-container-id",
				Name:        "stopped-container",
				Status:      "Exited (0)",
				CPUUsage:    0,
				MemoryUsage: 0,
			},
		},
		CPUUsage:     25,
		MemoryUsage:  50,
		StorageUsage: 75,
		Uptime:       "1D, 2H, 3M, 4S",
	}

	// Parse the dashboard template.
	tmpl, err := template.ParseFiles(
		"web/index.html",
		"web/containers.html",
		"web/performance.html",
	)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	// Execute the template into a response recorder.
	rr := httptest.NewRecorder()
	if err := tmpl.Execute(rr, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	// Verify that the template rendered successfully.
	if rr.Code != 200 {
		t.Errorf("expected status code 200, got %d", rr.Code)
	}

	body := rr.Body.String()

	// Verify that the expected dashboard data was rendered.
	expectedContent := []string{
		"Server Dashboard",
		"25%",
		"50%",
		"75%",
		"1D, 2H, 3M, 4S",
		"test-container",
		"Up",
		"10.5%",
		"20.25%",
		"stopped-container",
		"Exited (0)",
	}

	for _, content := range expectedContent {
		if !strings.Contains(body, content) {
			t.Errorf("expected response body to contain %q", content)
		}
	}
}

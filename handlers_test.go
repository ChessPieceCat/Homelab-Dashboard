package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"homelab-dashboard/internal/docker"
	"html/template"
)

func TestDashboardHandler(t *testing.T) {
	tmpl, err := template.ParseFiles(
		"web/index.html",
		"web/containers.html",
		"web/performance.html",
	)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	monitor := docker.NewMonitor(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler := dashboardHandler(monitor, tmpl)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()

	expectedContent := []string{
		"Server Dashboard",
		"Containers",
		"CPU:",
		"Memory:",
		"Storage:",
		"Uptime:",
	}

	for _, content := range expectedContent {
		if !strings.Contains(body, content) {
			t.Errorf(
				"expected response body to contain %q",
				content,
			)
		}
	}
}

func TestContainerActionHandlerMethodNotAllowed(t *testing.T) {
	handler := containerActionHandler(nil, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/container/start",
		nil,
	)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rr.Code,
		)
	}
}

func TestContainerActionHandlerMissingContainerID(t *testing.T) {
	handler := containerActionHandler(nil, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/container/start",
		nil,
	)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rr.Code,
		)
	}
}

func TestContainerActionHandlerUnknownAction(t *testing.T) {
	handler := containerActionHandler(nil, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/container/delete",
		strings.NewReader("containerID=test-container"),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rr.Code,
		)
	}

	if !strings.Contains(rr.Body.String(), "Unknown action") {
		t.Errorf(
			"expected response to contain %q, got %q",
			"Unknown action",
			rr.Body.String(),
		)
	}
}

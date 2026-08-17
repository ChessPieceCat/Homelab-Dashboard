package main

import (
	"homelab-dashboard/internal/docker"
	"html/template"
	"log"
	"net/http"
	"time"
)

func main() {
	// Create a Docker client
	dockerClient, err := docker.CreateDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Serve static files from the "web" directory
	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("web")),
		),
	)

	// Create a new monitor instance
	monitor := docker.NewMonitor(dockerClient)

	// Start the monitor
	monitor.Start(5 * time.Second) // Update every 5 seconds

	tmpl, err := template.ParseFiles(
		"web/index.html",
		"web/containers.html",
		"web/performance.html",
	)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	http.HandleFunc("/", dashboardHandler(monitor, tmpl))
	http.HandleFunc("/containers", containersHandler(monitor, tmpl))
	http.HandleFunc("/performance", performanceHandler(tmpl))
	http.HandleFunc("/container/start", containerActionHandler(dockerClient, monitor))
	http.HandleFunc("/container/stop", containerActionHandler(dockerClient, monitor))
	http.HandleFunc("/container/restart", containerActionHandler(dockerClient, monitor))
	http.HandleFunc("/container/delete", containerActionHandler(dockerClient, monitor))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

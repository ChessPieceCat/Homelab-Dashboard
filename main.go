package main

import (
	"homelab-dashboard/internal/docker"
	"homelab-dashboard/internal/system"
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

	tmpl, err := template.ParseFiles("web/index.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Serve the index.html file
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get container statuses
		statuses := monitor.GetStatuses()

		// Get system usage
		cpuUsage, memoryUsage, storageUsage, err := system.GetSystemUsage()
		if err != nil {
			log.Printf("Failed to get system usage: %v", err)
			http.Error(w, "Failed to get system usage", http.StatusInternalServerError)
			return
		}

		// Get system uptime
		uptime, err := system.GetSystemUptime()
		if err != nil {
			log.Printf("Failed to get system uptime: %v", err)
			http.Error(w, "Failed to get system uptime", http.StatusInternalServerError)
			return
		}

		// Format uptime
		uptimeStr := system.FormatUptime(uptime)

		// Pass the container statuses and system usage to the template
		data := DashboardData{
			Statuses:     statuses,
			CPUUsage:     cpuUsage,
			MemoryUsage:  memoryUsage,
			StorageUsage: storageUsage,
			Uptime:       uptimeStr,
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

type DashboardData struct {
	Statuses     []docker.Container
	CPUUsage     float64
	MemoryUsage  float64
	StorageUsage float64
	Uptime       string
}

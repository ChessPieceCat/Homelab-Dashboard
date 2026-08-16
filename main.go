package main

import (
	"homelab-dashboard/internal/docker"
	"homelab-dashboard/internal/system"
	"html/template"
	"log"
	"net/http"
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

	// Serve static files from the "web" directory
	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("web")),
		),
	)

	// Serve the index.html file
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("web/index.html")
		if err != nil {
			http.Error(w, "Failed to load template", http.StatusInternalServerError)
			return
		}

		// Get container statuses
		statuses, err := docker.GetContainerStatuses(dockerClient)
		if err != nil {
			log.Printf("Failed to get container statuses: %v", err)
			http.Error(w, "Failed to get container statuses", http.StatusInternalServerError)
			return
		}

		// Get system usage
		cpuUsage, memoryUsage, storageUsage, err := system.GetSystemUsage()
		if err != nil {
			log.Printf("Failed to get system usage: %v", err)
			http.Error(w, "Failed to get system usage", http.StatusInternalServerError)
			return
		}

		// Pass the container statuses and system usage to the template
		data := struct {
			Statuses     []docker.Container
			CPUUsage     float64
			MemoryUsage  float64
			StorageUsage float64
		}{
			Statuses:     statuses,
			CPUUsage:     cpuUsage,
			MemoryUsage:  memoryUsage,
			StorageUsage: storageUsage,
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

package main

import (
	"homelab-dashboard/internal/docker"
	"homelab-dashboard/internal/system"
	"html/template"
	"log"
	"net/http"
	"path"

	"github.com/moby/moby/client"
)

func dashboardHandler(monitor *docker.Monitor, tmpl *template.Template) http.HandlerFunc {
	// Serve the index.html file
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func containerActionHandler(dockerClient *client.Client, monitor *docker.Monitor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		containerID := r.FormValue("containerID")
		if containerID == "" {
			http.Error(w, "Missing container ID", http.StatusBadRequest)
			return
		}

		// Get the container action from the URL path.
		action := path.Base(r.URL.Path)

		var err error
		switch action {
		case "start":
			log.Printf("Starting container: %s", containerID)
			_, err = dockerClient.ContainerStart(r.Context(), containerID, client.ContainerStartOptions{})
		case "stop":
			log.Printf("Stopping container: %s", containerID)
			_, err = dockerClient.ContainerStop(r.Context(), containerID, client.ContainerStopOptions{})
		case "restart":
			log.Printf("Restarting container: %s", containerID)
			_, err = dockerClient.ContainerRestart(r.Context(), containerID, client.ContainerRestartOptions{})
		default:
			http.Error(w, "Unknown action: "+action, http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, "Failed to "+action+" container: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := monitor.RefreshContainer(containerID); err != nil {
			log.Printf("Failed to refresh container %s: %v", containerID, err)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

type DashboardData struct {
	Statuses     []docker.Container
	CPUUsage     float64
	MemoryUsage  float64
	StorageUsage float64
	Uptime       string
}

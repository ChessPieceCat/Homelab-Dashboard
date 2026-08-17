package main

import (
	"homelab-dashboard/internal/docker"
	"homelab-dashboard/internal/system"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/moby/moby/client"
)

func dashboardHandler(monitor *docker.Monitor, tmpl *template.Template) http.HandlerFunc {
	// Serve the index.html file
	return func(w http.ResponseWriter, r *http.Request) {
		errorMessage := r.URL.Query().Get("error")

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
			Containers:   statuses,
			CPUUsage:     cpuUsage,
			MemoryUsage:  memoryUsage,
			StorageUsage: storageUsage,
			Uptime:       uptimeStr,
			ErrorMessage: errorMessage,
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

		switch action {
		case "start", "stop", "restart", "delete":
			// Valid action.
		default:
			http.Error(
				w,
				"Unknown action: "+action,
				http.StatusBadRequest,
			)
			return
		}

		containerActionMutex.Lock()
		defer containerActionMutex.Unlock()

		info, err := dockerClient.ContainerInspect(r.Context(), containerID, client.ContainerInspectOptions{})
		if err != nil {
			redirectWithError(
				w,
				r,
				"Failed to inspect container: "+err.Error(),
			)
			return
		}

		log.Printf(
			"Container %s state before %s: running=%t, status=%s",
			containerID,
			action,
			info.Container.State.Running,
			info.Container.State.Status,
		)

		switch action {
		case "start":
			status := info.Container.State.Status

			if status == "running" {
				redirectWithError(w, r, "Container is already running")
				return
			}

			if status == "restarting" || status == "stopping" {
				redirectWithError(
					w,
					r,
					"Container is currently "+string(status),
				)
				return
			}

			if status != "created" && status != "exited" {
				redirectWithError(
					w,
					r,
					"Container cannot be started from state: "+string(status),
				)
				return
			}

			log.Printf("Starting container: %s", containerID)

			_, err = dockerClient.ContainerStart(
				r.Context(),
				containerID,
				client.ContainerStartOptions{},
			)

		case "stop":
			status := info.Container.State.Status

			if status == "stopping" {
				redirectWithError(w, r, "Container is already stopping")
				return
			}

			if status != "running" {
				redirectWithError(w, r, "Container is not running")
				return
			}

			log.Printf("Stopping container: %s", containerID)

			_, err = dockerClient.ContainerStop(
				r.Context(),
				containerID,
				client.ContainerStopOptions{},
			)
		case "restart":
			log.Printf("Restarting container: %s", containerID)
			_, err = dockerClient.ContainerRestart(r.Context(), containerID, client.ContainerRestartOptions{})
		case "delete":
			log.Printf("Deleting container: %s", containerID)

			_, err = dockerClient.ContainerRemove(
				r.Context(),
				containerID,
				client.ContainerRemoveOptions{
					Force:         false,
					RemoveVolumes: false,
					RemoveLinks:   false,
				},
			)

			if err != nil {
				redirectWithError(
					w,
					r,
					"Failed to "+action+" container: "+err.Error(),
				)
				return
			}

			if err := monitor.RemoveContainer(containerID); err != nil {
				log.Printf(
					"Failed to remove container %s from monitor: %v",
					containerID,
					err,
				)
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return

		default:
			http.Error(w, "Unknown action: "+action, http.StatusBadRequest)
			return
		}

		if err != nil {
			redirectWithError(
				w,
				r,
				"Failed to "+action+" container: "+err.Error(),
			)
			return
		}

		if err := monitor.RefreshContainer(containerID); err != nil {
			log.Printf("Failed to refresh container %s: %v", containerID, err)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func containersHandler(monitor *docker.Monitor, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses := monitor.GetStatuses()

		data := DashboardData{
			Containers: statuses,
		}

		if err := tmpl.ExecuteTemplate(w, "containers", data); err != nil {
			http.Error(
				w,
				"Failed to render containers: "+err.Error(),
				http.StatusInternalServerError,
			)
		}
	}
}

func performanceHandler(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cpuUsage, memoryUsage, storageUsage, err := system.GetSystemUsage()
		if err != nil {
			log.Printf("Failed to get system usage: %v", err)
			http.Error(
				w,
				"Failed to get system usage",
				http.StatusInternalServerError,
			)
			return
		}

		uptime, err := system.GetSystemUptime()
		if err != nil {
			log.Printf("Failed to get system uptime: %v", err)
			http.Error(
				w,
				"Failed to get system uptime",
				http.StatusInternalServerError,
			)
			return
		}

		data := DashboardData{
			CPUUsage:     cpuUsage,
			MemoryUsage:  memoryUsage,
			StorageUsage: storageUsage,
			Uptime:       system.FormatUptime(uptime),
		}

		if err := tmpl.ExecuteTemplate(w, "performance", data); err != nil {
			http.Error(
				w,
				"Failed to render performance: "+err.Error(),
				http.StatusInternalServerError,
			)
		}
	}
}

func redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(
		w,
		r,
		"/?error="+url.QueryEscape(message),
		http.StatusSeeOther,
	)
}

type DashboardData struct {
	Containers   []docker.Container
	CPUUsage     float64
	MemoryUsage  float64
	StorageUsage float64
	Uptime       string
	ErrorMessage string
}

var containerActionMutex sync.Mutex

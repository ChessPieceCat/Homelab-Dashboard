package docker

import (
	"context"
	"encoding/json"
	"log"
	"math"

	"github.com/moby/moby/client"
)

func CreateDockerClient() (*client.Client, error) {
	apiClient, err := client.New(
		client.FromEnv,
		client.WithUserAgent("homelab-dashboard"),
	)

	if err != nil {
		return nil, err
	}

	return apiClient, nil
}

// Return each container's name and status
func GetContainerStatuses(apiClient *client.Client) ([]Container, error) {
	containers, err := apiClient.ContainerList(context.Background(), client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var containerStatuses []Container
	for _, container := range containers.Items {
		containerStatuses = append(containerStatuses, Container{
			Name:         container.Names[0],
			Status:       container.Status,
			CPUUsage:     0,
			MemoryUsage:  0,
			StorageUsage: 0,
		})
	}

	// Get container resource usage and update the containerStatuses slice
	updatedStatuses, err := getContainerResourceUsage(apiClient, containerStatuses)
	if err != nil {
		log.Printf("Failed to get container resource usage: %v", err)
		return nil, err
	}

	return updatedStatuses, nil
}

// Get container CPU and memory usage percentages and add to the Container struct
func getContainerResourceUsage(dockerClient *client.Client, statuses []Container) ([]Container, error) {
	var updatedStatuses []Container

	for _, container := range statuses {
		usage, err := dockerClient.ContainerStats(context.Background(), container.Name, client.ContainerStatsOptions{Stream: false})
		if err != nil {
			log.Printf("Failed to get resource usage for container %s: %v", container.Name, err)
			continue
		}
		defer usage.Body.Close()

		// used_memory = memory_stats.usage - memory_stats.stats.cache (cgroups v1)
		// used_memory = memory_stats.usage - memory_stats.stats.inactive_file (cgroups v2)
		// available_memory = memory_stats.limit
		// Memory usage % = (used_memory / available_memory) * 100.0
		// cpu_delta = cpu_stats.cpu_usage.total_usage - precpu_stats.cpu_usage.total_usage
		// system_cpu_delta = cpu_stats.system_cpu_usage - precpu_stats.system_cpu_usage
		// number_cpus = length(cpu_stats.cpu_usage.percpu_usage) or cpu_stats.online_cpus
		// CPU usage % = (cpu_delta / system_cpu_delta) * number_cpus * 100.0

		// Decode the JSON response into a struct
		var stats struct {
			CPUStats struct {
				CPUUsage struct {
					TotalUsage  uint64   `json:"total_usage"`
					PercpuUsage []uint64 `json:"percpu_usage"`
				} `json:"cpu_usage"`
				SystemCPUUsage uint64 `json:"system_cpu_usage"`
				OnlineCPUs     uint32 `json:"online_cpus"`
			} `json:"cpu_stats"`
			PreCPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemCPUUsage uint64 `json:"system_cpu_usage"`
			} `json:"precpu_stats"`
			MemoryStats struct {
				Usage uint64            `json:"usage"`
				Limit uint64            `json:"limit"`
				Stats map[string]uint64 `json:"stats"`
			} `json:"memory_stats"`
		}

		if err := json.NewDecoder(usage.Body).Decode(&stats); err != nil {
			log.Printf("Failed to decode resource usage for container %s: %v", container.Name, err)
			continue
		}

		// Calculate CPU usage percentage
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		// If cpuDelta is zero, skip calculation and set CPU usage to 0 to avoid division by zero
		if cpuDelta == 0 {
			container.CPUUsage = 0
			updatedStatuses = append(updatedStatuses, container)
			continue
		}
		systemCPUDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
		numCPUs := float64(stats.CPUStats.OnlineCPUs)
		cpuUsagePercent := (cpuDelta / systemCPUDelta) * numCPUs * 100.0

		// Calculate Memory usage percentage
		var usedMemory uint64
		if _, ok := stats.MemoryStats.Stats["cache"]; ok {
			usedMemory = stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
		} else if _, ok := stats.MemoryStats.Stats["inactive_file"]; ok {
			usedMemory = stats.MemoryStats.Usage - stats.MemoryStats.Stats["inactive_file"]
		} else {
			usedMemory = stats.MemoryStats.Usage
		}
		memoryUsagePercent := (float64(usedMemory) / float64(stats.MemoryStats.Limit)) * 100.0

		// Update the container struct with resource usage
		container.CPUUsage = math.Round(cpuUsagePercent*100) / 100
		container.MemoryUsage = math.Round(memoryUsagePercent*100) / 100

		updatedStatuses = append(updatedStatuses, container)
	}

	return updatedStatuses, nil
}

// Create containers struct to hold container information
type Container struct {
	Name         string
	Status       string
	CPUUsage     float64
	MemoryUsage  float64
	StorageUsage float64
}

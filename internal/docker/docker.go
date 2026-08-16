package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

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
func GetContainers(apiClient *client.Client) ([]Container, error) {
	containers, err := apiClient.ContainerList(context.Background(), client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var containerStatuses []Container
	for _, container := range containers.Items {
		containerStatuses = append(containerStatuses, Container{
			ID:          container.ID,
			Name:        container.Names[0],
			Status:      container.Status,
			CPUUsage:    0,
			MemoryUsage: 0,
		})
	}

	// Get container resource usage and update the containerStatuses slice
	updatedStatuses, err := getContainerResourceUsage(apiClient, containerStatuses)
	if err != nil {
		log.Printf("Failed to get container resource usage: %v", err)
		return nil, err
	}

	sort.Slice(updatedStatuses, func(i, j int) bool {
		return updatedStatuses[i].Name < updatedStatuses[j].Name
	})

	return updatedStatuses, nil
}

// containerStats mirrors the subset of the Docker stats JSON response we need.
type containerStats struct {
	CPUStats    cpuStats `json:"cpu_stats"`
	PreCPUStats cpuStats `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
}

// calculateCPUUsagePercent computes CPU usage % from two stats samples.
// cpu_delta = current CPU usage - previous CPU usage
// system_cpu_delta = current system CPU usage - previous system CPU usage
// number_cpus = number of online CPUs
// CPU usage % = (cpu_delta / system_cpu_delta) * 100.0
func calculateCPUUsagePercent(previous, current cpuStats) float64 {
	if current.CPUUsage.TotalUsage < previous.CPUUsage.TotalUsage {
		return 0
	}

	if current.SystemCPUUsage < previous.SystemCPUUsage {
		return 0
	}

	cpuDelta := float64(
		current.CPUUsage.TotalUsage -
			previous.CPUUsage.TotalUsage,
	)

	systemCPUDelta := float64(
		current.SystemCPUUsage -
			previous.SystemCPUUsage,
	)

	if cpuDelta == 0 || systemCPUDelta == 0 {
		return 0
	}

	// numCPUs := float64(current.CPUStats.OnlineCPUs)
	// if numCPUs == 0 {
	// 	return 0
	// }

	cpuUsagePercent := (cpuDelta / systemCPUDelta) * 100.0

	return math.Round(cpuUsagePercent*100) / 100
}

// calculateMemoryUsagePercent computes memory usage % from a decoded stats payload.
// used_memory = memory_stats.usage - memory_stats.stats.cache (cgroups v1)
// used_memory = memory_stats.usage - memory_stats.stats.inactive_file (cgroups v2)
// available_memory = memory_stats.limit
// Memory usage % = (used_memory / available_memory) * 100.0
func calculateMemoryUsagePercent(stats containerStats) float64 {
	var usedMemory uint64
	if cache, ok := stats.MemoryStats.Stats["cache"]; ok {
		if cache > stats.MemoryStats.Usage {
			return 0
		}
		usedMemory = stats.MemoryStats.Usage - cache
	} else if inactiveFile, ok := stats.MemoryStats.Stats["inactive_file"]; ok {
		if inactiveFile > stats.MemoryStats.Usage {
			return 0
		}
		usedMemory = stats.MemoryStats.Usage - inactiveFile
	} else {
		usedMemory = stats.MemoryStats.Usage
	}

	if stats.MemoryStats.Limit == 0 {
		return 0
	}

	memoryUsagePercent := (float64(usedMemory) / float64(stats.MemoryStats.Limit)) * 100.0

	return math.Round(memoryUsagePercent*100) / 100
}

// getContainerResourceUsage fetches CPU and memory usage percentages for each
// container and returns an updated slice with those fields populated
func getContainerResourceUsage(
	dockerClient *client.Client,
	statuses []Container,
) ([]Container, error) {

	results := make(chan Container, len(statuses))

	for _, container := range statuses {
		go func(container Container) {

			// Stopped containers don't have live resource statistics.
			if !strings.HasPrefix(container.Status, "Up") {
				results <- container
				return
			}

			ctx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			usage, err := dockerClient.ContainerStats(
				ctx,
				container.Name,
				client.ContainerStatsOptions{
					Stream:                false,
					IncludePreviousSample: true,
				},
			)
			if err != nil {
				log.Printf(
					"Failed to get resource usage for container %s: %v",
					container.Name,
					err,
				)

				results <- container
				return
			}

			defer usage.Body.Close()

			var stats containerStats

			if err := json.NewDecoder(usage.Body).Decode(&stats); err != nil {
				log.Printf(
					"Failed to decode resource usage for container %s: %v",
					container.Name,
					err,
				)

				results <- container
				return
			}

			container.CPUUsage = calculateCPUUsagePercent(
				stats.PreCPUStats,
				stats.CPUStats,
			)

			container.MemoryUsage = calculateMemoryUsagePercent(stats)

			results <- container
		}(container)
	}

	var updatedStatuses []Container

	for range statuses {
		container := <-results
		updatedStatuses = append(updatedStatuses, container)
	}

	return updatedStatuses, nil
}

func GetContainerStatus(
	dockerClient *client.Client,
	containerID string,
) (string, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	containers, err := dockerClient.ContainerList(
		ctx,
		client.ContainerListOptions{
			All: true,
		},
	)
	if err != nil {
		return "", err
	}

	for _, container := range containers.Items {
		if container.ID == containerID {
			return container.Status, nil
		}
	}

	return "", fmt.Errorf("container %s not found", containerID)
}

// Create containers struct to hold container information
type Container struct {
	ID          string
	Name        string
	Status      string
	CPUUsage    float64
	MemoryUsage float64
}

type cpuStats struct {
	CPUUsage struct {
		TotalUsage uint64 `json:"total_usage"`
	} `json:"cpu_usage"`
	SystemCPUUsage uint64 `json:"system_cpu_usage"`
}

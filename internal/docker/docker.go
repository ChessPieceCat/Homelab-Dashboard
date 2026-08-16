package docker

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"sync"
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

	return updatedStatuses, nil
}

// containerStats mirrors the subset of the Docker stats JSON response we need.
type containerStats struct {
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

// calculateCPUUsagePercent computes CPU usage % from two stats samples.
// cpu_delta = current CPU usage - previous CPU usage
// system_cpu_delta = current system CPU usage - previous system CPU usage
// number_cpus = number of online CPUs
// CPU usage % = (cpu_delta / system_cpu_delta) * number_cpus * 100.0
func calculateCPUUsagePercent(previous, current containerStats) float64 {
	if current.CPUStats.CPUUsage.TotalUsage < previous.CPUStats.CPUUsage.TotalUsage {
		return 0
	}

	if current.CPUStats.SystemCPUUsage < previous.CPUStats.SystemCPUUsage {
		return 0
	}

	cpuDelta := float64(
		current.CPUStats.CPUUsage.TotalUsage -
			previous.CPUStats.CPUUsage.TotalUsage,
	)

	systemCPUDelta := float64(
		current.CPUStats.SystemCPUUsage -
			previous.CPUStats.SystemCPUUsage,
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
		usedMemory = stats.MemoryStats.Usage - cache
	} else if inactiveFile, ok := stats.MemoryStats.Stats["inactive_file"]; ok {
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
func getContainerResourceUsage(dockerClient *client.Client, statuses []Container) ([]Container, error) {
	results := make(chan Container)

	for _, container := range statuses {
		go func(container Container) {
			usage, err := dockerClient.ContainerStats(
				context.Background(),
				container.Name,
				client.ContainerStatsOptions{Stream: true},
			)
			if err != nil {
				log.Printf(
					"Failed to get resource usage for container %s: %v",
					container.Name,
					err,
				)

				// Send the container back with zero resource usage.
				results <- container
				return
			}

			defer usage.Body.Close()

			decoder := json.NewDecoder(usage.Body)

			var previousStats containerStats
			var currentStats containerStats

			// Get the first stats sample.
			if err := decoder.Decode(&previousStats); err != nil {
				log.Printf(
					"Failed to decode initial resource usage for container %s: %v",
					container.Name,
					err,
				)

				results <- container
				return
			}

			// Get the second stats sample.
			if err := decoder.Decode(&currentStats); err != nil {
				log.Printf(
					"Failed to decode second resource usage for container %s: %v",
					container.Name,
					err,
				)

				results <- container
				return
			}

			container.CPUUsage = calculateCPUUsagePercent(
				previousStats,
				currentStats,
			)

			container.MemoryUsage = calculateMemoryUsagePercent(currentStats)

			results <- container
		}(container)
	}

	// Collect one result from each goroutine.
	var updatedStatuses []Container

	for range statuses {
		container := <-results
		updatedStatuses = append(updatedStatuses, container)
	}

	return updatedStatuses, nil
}

func (m *Monitor) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			m.update()

			<-ticker.C
		}
	}()
}

func (m *Monitor) update() {
	// Get container statuses
	statuses, err := GetContainerStatuses(m.client)
	if err != nil {
		log.Printf("Failed to get container statuses: %v", err)
		return
	}

	m.mutex.Lock()
	m.statuses = statuses
	m.mutex.Unlock()
}

func (m *Monitor) GetStatuses() []Container {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy of the statuses slice to avoid race conditions
	statusesCopy := make([]Container, len(m.statuses))
	copy(statusesCopy, m.statuses)

	return statusesCopy
}

// NewMonitor creates a new Monitor instance with the provided Docker client.
func NewMonitor(client *client.Client) *Monitor {
	return &Monitor{
		client:   client,
		statuses: []Container{},
		mutex:    sync.RWMutex{},
	}
}

// Create containers struct to hold container information
type Container struct {
	Name        string
	Status      string
	CPUUsage    float64
	MemoryUsage float64
}

type Monitor struct {
	client   *client.Client
	statuses []Container
	mutex    sync.RWMutex
}

package docker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/moby/moby/client"
)

func (m *Monitor) Start(interval time.Duration) {
	go func() {
		m.Update() // Initial update before starting the ticker

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.Update()
		}
	}()
}

func (m *Monitor) Update() {
	// Get container statuses
	statuses, err := GetContainers(m.client)
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

func (m *Monitor) RefreshContainer(containerID string) error {
	status, err := GetContainerStatus(m.client, containerID)
	if err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i := range m.statuses {
		if m.statuses[i].ID == containerID {
			m.statuses[i].Status = status

			// If the container is no longer running,
			// its resource usage is no longer meaningful.
			if status != "running" {
				m.statuses[i].CPUUsage = 0
				m.statuses[i].MemoryUsage = 0
			}

			return nil
		}
	}

	return fmt.Errorf("container %s not found in monitor", containerID)
}

func (m *Monitor) RemoveContainer(containerID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i := range m.statuses {
		if m.statuses[i].ID == containerID {
			// Remove the container from the slice
			m.statuses = append(m.statuses[:i], m.statuses[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("container %s not found in monitor", containerID)
}

// NewMonitor creates a new Monitor instance with the provided Docker client.
func NewMonitor(client *client.Client) *Monitor {
	return &Monitor{
		client: client,
	}
}

type Monitor struct {
	client   *client.Client
	statuses []Container
	mutex    sync.RWMutex
}

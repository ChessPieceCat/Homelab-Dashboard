package main

import (
	"context"

	"github.com/moby/moby/client"
)

func createDockerClient() (*client.Client, error) {
	apiClient, err := client.New(
		client.FromEnv,
		client.WithUserAgent("homelab-dashboard"),
	)

	if err != nil {
		return nil, err
	}

	return apiClient, nil
}

// Return each container's name and status as a map
func getContainerStatuses(apiClient *client.Client) ([]Container, error) {
	containers, err := apiClient.ContainerList(context.Background(), client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var containerStatuses []Container
	for _, container := range containers.Items {
		containerStatuses = append(containerStatuses, Container{
			Name:   container.Names[0],
			Status: container.Status,
		})
	}

	return containerStatuses, nil
}

// Create containers struct to hold container information
type Container struct {
	Name   string
	Status string
}

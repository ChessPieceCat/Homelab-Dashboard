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
func getContainerStatuses(apiClient *client.Client) (map[string]string, error) {
	containers, err := apiClient.ContainerList(context.Background(), client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	statuses := make(map[string]string)
	for _, container := range containers.Items {
		statuses[container.Names[0]] = container.Status
	}

	return statuses, nil
}

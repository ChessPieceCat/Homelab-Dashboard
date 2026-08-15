package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Create a Docker client
	dockerClient, err := createDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Get container statuses
	statuses, err := getContainerStatuses(dockerClient)
	if err != nil {
		log.Fatalf("Failed to get container statuses: %v", err)
	}

	// Log container statuses
	for _, status := range statuses {
		log.Printf("Container: %s, Status: %s", status.Name, status.Status)
	}

	// Serve the index.html file
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("web/index.html")
		if err != nil {
			log.Fatalf("Failed to read index.html: %v", err)
		}
		w.Write(file)
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

package main

import (
	"html/template"
	"log"
	"net/http"
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

		// Pass the container statuses to the template
		data := struct {
			Statuses []Container
		}{
			Statuses: statuses,
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("/home/tum/Documents/Homelab Dashboard/web/templates/index.html")
		if err != nil {
			log.Fatalf("Failed to read index.html: %v", err)
		}
		w.Write(file)
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

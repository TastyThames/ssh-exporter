package main

import (
	"log"
	"net/http"
	"os"

	"github.com/tastythames/ssh-exporter/internal/inventory"
	)


func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	listen := getenv("EXPORTER_LISTEN", ":9222")
	invPath := getenv("INVENTORY_PATH", "deploy/targets.yaml")

	inv, err := inventory.Load(invPath)
	if err != nil {
		log.Fatalf("inventory load failed: %v", err)
	}
	targets, err := inv.Normalize()
	if err != nil {
		log.Fatalf("inventory normalize failed: %v", err)
	}
	log.Printf("inventory loaded: %d targets from %s", len(targets), invPath)
	for _, t := range targets {
		log.Printf("target: name=%s addr=%s mode=%s labels=%v", t.Name, t.Address, t.Mode, t.Labels)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// NOTE: /metrics will be added next (with client_golang registry)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/health", http.StatusFound)
	})

	log.Printf("ssh-agentless-exporter listening on %s", listen)
	log.Fatal(http.ListenAndServe(listen, mux))
}

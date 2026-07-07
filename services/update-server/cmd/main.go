// Command update-server serves signed OTA manifests and receives kiosk health reports.
package main

import (
	"log"
	"net/http"

	"github.com/astra-service/update-server/internal/config"
	"github.com/astra-service/update-server/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("update-server: config error: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("update-server: init error: %v", err)
	}

	addr := ":" + cfg.Port
	log.Printf("update-server listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("update-server: serve error: %v", err)
	}
}

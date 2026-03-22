package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stratogroup/neuroforge-sdx/tools/compliance/docsserver"
)

func main() {
	server := &http.Server{
		Addr:              docsListenAddr(),
		Handler:           docsserver.NewHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("serving compliance docs on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func docsListenAddr() string {
	if addr := strings.TrimSpace(os.Getenv("COMPLIANCE_DOCS_ADDR")); addr != "" {
		return addr
	}

	if port := strings.TrimSpace(os.Getenv("COMPLIANCE_DOCS_PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}

	return ":8091"
}

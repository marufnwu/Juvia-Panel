package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"panel-api/internal/agent"
)

func main() {
	// Determine socket path
	socketPath := os.Getenv("PANEL_AGENT_SOCKET")
	if socketPath == "" {
		socketPath = "/var/run/panel/agent.sock"
	}

	// Create agent
	a := agent.New(socketPath)

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start agent
	log.Printf("Starting agent daemon on %s", socketPath)
	if err := a.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	log.Printf("Agent daemon is running")

	// Wait for shutdown signal
	<-quit
	log.Println("Shutting down agent daemon...")

	// Stop agent
	if err := a.Stop(); err != nil {
		log.Fatalf("Failed to stop agent: %v", err)
	}

	log.Println("Agent daemon stopped")
}
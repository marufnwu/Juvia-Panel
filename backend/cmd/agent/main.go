package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"panel-api/internal/agent"
	"panel-api/internal/config"
)

func main() {
	// Load configuration (supports --config and --version flags)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Determine socket path from config
	socketPath := cfg.AgentSocket
	if socketPath == "" {
		socketPath = config.DefaultAgentSocket
	}

	// Create agent
	a := agent.New(socketPath)

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start agent
	log.Printf("Starting agent daemon (version %s) on %s", cfg.Version, socketPath)
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

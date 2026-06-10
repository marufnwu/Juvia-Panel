package services

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"panel-api/internal/agent"
	"panel-api/internal/database"
)

// AppReconciler periodically checks container state and updates the database.
type AppReconciler struct {
	db         *database.DB
	agent      *agent.Client
	interval   time.Duration
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewAppReconciler creates a new reconciler.
func NewAppReconciler(db *database.DB, agentClient *agent.Client, interval time.Duration) *AppReconciler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &AppReconciler{
		db:       db,
		agent:    agentClient,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic reconciliation.
func (r *AppReconciler) Start() {
	r.wg.Add(1)
	go r.run()
	log.Printf("App reconciler started (interval: %s)", r.interval)
}

// Stop stops periodic reconciliation.
func (r *AppReconciler) Stop() {
	close(r.stopChan)
	r.wg.Wait()
	log.Println("App reconciler stopped")
}

func (r *AppReconciler) run() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Run once immediately
	r.reconcile()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.reconcile()
		}
	}
}

func (r *AppReconciler) reconcile() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := r.agent.ListContainers(ctx)
	if err != nil {
		log.Printf("Reconciler: failed to list containers: %v", err)
		return
	}

	containerMap := make(map[string]*agent.ContainerInfo)
	for _, c := range containers {
		if strings.HasPrefix(c.Name, "panel-") {
			appID := strings.TrimPrefix(c.Name, "panel-")
			containerMap[appID] = c
		}
	}

	var runningApps []database.App
	appsQuery := `SELECT id, status, container_id FROM apps WHERE status = 'running'`
	if err := r.db.SelectContext(ctx, &runningApps, appsQuery); err != nil {
		log.Printf("Reconciler: failed to query running apps: %v", err)
		return
	}

	for _, app := range runningApps {
		ci, ok := containerMap[app.ID]
		if !ok {
			log.Printf("Reconciler: app %s has no container, marking as failed", app.ID)
			r.db.ExecContext(ctx, "UPDATE apps SET status = 'failed' WHERE id = ?", app.ID)
			continue
		}
		if ci.State != "running" {
			log.Printf("Reconciler: app %s container is %s (exit code %d), marking as failed", app.ID, ci.State, ci.ExitCode)
			r.db.ExecContext(ctx, "UPDATE apps SET status = 'failed' WHERE id = ?", app.ID)
		}
	}
}

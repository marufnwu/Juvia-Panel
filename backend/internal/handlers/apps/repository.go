package apps

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"panel-api/internal/database"
)

// AppRepository handles database operations for apps
type AppRepository struct {
	db *database.DB
}

// NewAppRepository creates a new app repository
func NewAppRepository(db *database.DB) *AppRepository {
	return &AppRepository{db: db}
}

// ListAppsParams holds parameters for listing apps
type ListAppsParams struct {
	Status  string
	Runtime string
	Search  string
	Sort    string
	Order   string
	Page    int
	PerPage int
}

// ListApps returns a paginated list of apps
func (r *AppRepository) ListApps(ctx context.Context, params ListAppsParams) ([]database.AppListItem, int, error) {
	// Build query with filters - optimized with LEFT JOINs to avoid N+1 queries
	query := `SELECT a.*, 
		(SELECT domain FROM app_domains WHERE app_id = a.id AND is_primary = 1 LIMIT 1) as primary_domain,
		COALESCE((SELECT COUNT(*) FROM app_env_vars WHERE app_id = a.id), 0) as env_count,
		COALESCE((SELECT COUNT(*) FROM app_volumes WHERE app_id = a.id), 0) as volume_count,
		COALESCE((SELECT completed_at FROM deployments WHERE app_id = a.id AND status = 'success' ORDER BY completed_at DESC LIMIT 1), '') as last_deployed_at
		FROM apps a WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM apps a WHERE 1=1`
	var args []interface{}
	
	// Apply filters
	if params.Status != "" && params.Status != "all" {
		query += " AND a.status = ?"
		countQuery += " AND a.status = ?"
		args = append(args, params.Status)
	}
	
	if params.Runtime != "" {
		query += " AND a.runtime = ?"
		countQuery += " AND a.runtime = ?"
		args = append(args, params.Runtime)
	}
	
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query += " AND (a.name LIKE ? OR EXISTS (SELECT 1 FROM app_domains d WHERE d.app_id = a.id AND d.domain LIKE ?))"
		countQuery += " AND (a.name LIKE ? OR EXISTS (SELECT 1 FROM app_domains d WHERE d.app_id = a.id AND d.domain LIKE ?))"
		args = append(args, searchPattern, searchPattern)
	}
	
	// Get total count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count apps: %w", err)
	}
	
	// Apply sorting
	validSorts := map[string]string{
		"name":       "a.name",
		"updated_at": "a.updated_at",
		"status":     "a.status",
	}
	sortField := validSorts[params.Sort]
	if sortField == "" {
		sortField = "a.updated_at"
	}
	
	order := "DESC"
	if params.Order == "asc" {
		order = "ASC"
	}
	
	query += fmt.Sprintf(" ORDER BY %s %s", sortField, order)
	
	// Apply pagination
	offset := (params.Page - 1) * params.PerPage
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.PerPage, offset)
	
	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list apps: %w", err)
	}
	defer rows.Close()
	
	var apps []database.AppListItem
	for rows.Next() {
		var app database.App
		var primaryDomain sql.NullString
		var envCount int
		var volumeCount int
		var lastDeployedAt sql.NullString
		
		err := rows.Scan(
			&app.ID, &app.Name, &app.Status, &app.HealthStatus, &app.Runtime, &app.RuntimeVersion,
			&app.SourceType, &app.SourceConfig, &app.BuildStrategy, &app.BuildConfig,
			&app.ContainerID, &app.ContainerImage, &app.InternalPort, &app.RestartPolicy,
			&app.HealthCheckPath, &app.HealthCheckInterval, &app.HealthCheckTimeout, &app.HealthCheckRetries,
			&app.CPULimit, &app.MemoryLimitMB, &app.MemorySwapMB, &app.CreatedBy,
			&app.CreatedAt, &app.UpdatedAt, &primaryDomain, &envCount, &volumeCount, &lastDeployedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan app: %w", err)
		}
		
		// Get domains for this app - skip for now to avoid timeout, domains can be loaded per-app
		domains := []string{}
		
		// Parse source config
		var sourceConfig database.SourceConfig
		database.ParseJSONField(&app.SourceConfig, &sourceConfig)
		
		var lastDeployed *time.Time
		if lastDeployedAt.Valid && lastDeployedAt.String != "" {
			// Try parsing the timestamp string
			layouts := []string{
				"2006-01-02 15:04:05.999999999 -0700 MST",
				"2006-01-02 15:04:05.999999999 -0700",
				"2006-01-02 15:04:05.999999999",
				time.RFC3339,
				"2006-01-02 15:04:05",
			}
			for _, layout := range layouts {
				if t, err := time.Parse(layout, lastDeployedAt.String); err == nil {
					lastDeployed = &t
					break
				}
			}
		}
		
		item := database.AppListItem{
			ID:             app.ID,
			Name:           app.Name,
			Status:         app.Status,
			HealthStatus:   app.HealthStatus,
			Runtime:        app.Runtime,
			RuntimeVersion: app.RuntimeVersion,
			PrimaryDomain:  nil,
			Domains:        domains,
			Source:         sourceConfig,
			BuildStrategy:  app.BuildStrategy,
			ContainerID:    app.ContainerID,
			Ports: database.Ports{
				Internal: app.InternalPort,
				External: nil,
			},
			EnvCount:       envCount,
			VolumeCount:    volumeCount,
			ResourceUsage:  nil,
			LastDeployedAt: lastDeployed,
			CreatedAt:      app.CreatedAt,
			UpdatedAt:      app.UpdatedAt,
		}
		
		if primaryDomain.Valid {
			item.PrimaryDomain = &primaryDomain.String
		}
		
		apps = append(apps, item)
	}
	
	if apps == nil {
		apps = []database.AppListItem{}
	}
	
	return apps, total, nil
}

// GetAppByID returns an app by ID
func (r *AppRepository) GetAppByID(ctx context.Context, id string) (*database.App, error) {
	var app database.App
	err := r.db.GetContext(ctx, &app, "SELECT * FROM apps WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}
	return &app, nil
}

// GetAppDomains returns all domains for an app
func (r *AppRepository) GetAppDomains(ctx context.Context, appID string) ([]string, error) {
	var domains []string
	rows, err := r.db.QueryContext(ctx, "SELECT domain FROM app_domains WHERE app_id = ?", appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	
	if domains == nil {
		domains = []string{}
	}
	return domains, nil
}

// GetAppDomainsDetail returns detailed domain info for an app
func (r *AppRepository) GetAppDomainsDetail(ctx context.Context, appID string) ([]database.AppDomain, error) {
	var domains []database.AppDomain
	err := r.db.SelectContext(ctx, &domains, "SELECT * FROM app_domains WHERE app_id = ?", appID)
	if err != nil {
		return nil, err
	}
	if domains == nil {
		domains = []database.AppDomain{}
	}
	return domains, nil
}

// GetEnvCount returns the number of environment variables for an app
func (r *AppRepository) GetEnvCount(ctx context.Context, appID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM app_env_vars WHERE app_id = ?", appID)
	return count, err
}

// GetVolumeCount returns the number of volumes for an app
func (r *AppRepository) GetVolumeCount(ctx context.Context, appID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM app_volumes WHERE app_id = ?", appID)
	return count, err
}

// GetLastDeployedAt returns the timestamp of the last successful deployment
func (r *AppRepository) GetLastDeployedAt(ctx context.Context, appID string) (*time.Time, error) {
	var completedAt sql.NullTime
	err := r.db.GetContext(ctx, &completedAt, 
		"SELECT completed_at FROM deployments WHERE app_id = ? AND status = 'success' ORDER BY completed_at DESC LIMIT 1", appID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		return &completedAt.Time, nil
	}
	return nil, nil
}

// GetAppVolumes returns all volumes for an app
func (r *AppRepository) GetAppVolumes(ctx context.Context, appID string) ([]database.AppVolume, error) {
	var volumes []database.AppVolume
	err := r.db.SelectContext(ctx, &volumes, "SELECT * FROM app_volumes WHERE app_id = ?", appID)
	if err != nil {
		return nil, err
	}
	if volumes == nil {
		volumes = []database.AppVolume{}
	}
	return volumes, nil
}

// GetAppVolumeByID returns a volume by ID
func (r *AppRepository) GetAppVolumeByID(ctx context.Context, volumeID string) (*database.AppVolume, error) {
	var volume database.AppVolume
	err := r.db.GetContext(ctx, &volume, "SELECT * FROM app_volumes WHERE id = ?", volumeID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &volume, nil
}

// GetAppEnvVars returns all environment variables for an app
func (r *AppRepository) GetAppEnvVars(ctx context.Context, appID string) ([]database.AppEnvVar, error) {
	var envVars []database.AppEnvVar
	err := r.db.SelectContext(ctx, &envVars, "SELECT * FROM app_env_vars WHERE app_id = ?", appID)
	if err != nil {
		return nil, err
	}
	if envVars == nil {
		envVars = []database.AppEnvVar{}
	}
	return envVars, nil
}

// GetConnectedServices returns services connected to an app
func (r *AppRepository) GetConnectedServices(ctx context.Context, appID string) ([]database.ConnectedService, error) {
	query := `
		SELECT s.id, s.name, s.type 
		FROM services s 
		JOIN service_app_links l ON s.id = l.service_id 
		WHERE l.app_id = ?
	`
	var services []database.ConnectedService
	rows, err := r.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var svc database.ConnectedService
		if err := rows.Scan(&svc.ID, &svc.Name, &svc.Type); err != nil {
			return nil, err
		}
		services = append(services, svc)
	}
	
	if services == nil {
		services = []database.ConnectedService{}
	}
	return services, nil
}

// CreateApp creates a new app
func (r *AppRepository) CreateApp(ctx context.Context, app *database.App) error {
	query := `
		INSERT INTO apps (id, name, status, health_status, runtime, runtime_version, 
			source_type, source_config, build_strategy, build_config, container_id, container_image,
			internal_port, restart_policy, health_check_path, health_check_interval, 
			health_check_timeout, health_check_retries, cpu_limit, memory_limit_mb, memory_swap_mb,
			created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		app.ID, app.Name, app.Status, app.HealthStatus, app.Runtime, app.RuntimeVersion,
		app.SourceType, app.SourceConfig, app.BuildStrategy, app.BuildConfig,
		app.ContainerID, app.ContainerImage, app.InternalPort, app.RestartPolicy,
		app.HealthCheckPath, app.HealthCheckInterval, app.HealthCheckTimeout, app.HealthCheckRetries,
		app.CPULimit, app.MemoryLimitMB, app.MemorySwapMB, app.CreatedBy, app.CreatedAt, app.UpdatedAt,
	)
	
	return err
}

// CreateAppDomain creates a new domain entry for an app
func (r *AppRepository) CreateAppDomain(ctx context.Context, domain *database.AppDomain) error {
	query := `
		INSERT INTO app_domains (app_id, domain, is_primary, force_https, ssl_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		domain.AppID, domain.Domain, domain.IsPrimary, domain.ForceHTTPS, domain.SSLStatus, domain.CreatedAt,
	)
	return err
}

// UpdateApp updates an existing app
func (r *AppRepository) UpdateApp(ctx context.Context, app *database.App) error {
	query := `
		UPDATE apps SET 
			name = ?, status = ?, health_status = ?, runtime = ?, runtime_version = ?,
			source_type = ?, source_config = ?, build_strategy = ?, build_config = ?,
			container_id = ?, container_image = ?, internal_port = ?, restart_policy = ?,
			health_check_path = ?, health_check_interval = ?, health_check_timeout = ?, health_check_retries = ?,
			cpu_limit = ?, memory_limit_mb = ?, memory_swap_mb = ?, updated_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.ExecContext(ctx, query,
		app.Name, app.Status, app.HealthStatus, app.Runtime, app.RuntimeVersion,
		app.SourceType, app.SourceConfig, app.BuildStrategy, app.BuildConfig,
		app.ContainerID, app.ContainerImage, app.InternalPort, app.RestartPolicy,
		app.HealthCheckPath, app.HealthCheckInterval, app.HealthCheckTimeout, app.HealthCheckRetries,
		app.CPULimit, app.MemoryLimitMB, app.MemorySwapMB, time.Now(), app.ID,
	)
	
	return err
}

// UpdateAppStatus updates only the status of an app
func (r *AppRepository) UpdateAppStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE apps SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id)
	return err
}

// DeleteApp deletes an app and all related data
func (r *AppRepository) DeleteApp(ctx context.Context, id string) error {
	// Delete volumes first (volumes have FK to apps with CASCADE, but we need to check host paths)
	_, err := r.db.ExecContext(ctx, "DELETE FROM apps WHERE id = ?", id)
	return err
}

// AppExists checks if an app name already exists
func (r *AppRepository) AppExists(ctx context.Context, name string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM apps WHERE name = ?", name)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ValidateGitURL validates a Git repository URL
func ValidateGitURL(url string) bool {
	// Basic Git URL validation patterns
	patterns := []string{
		`^https?://`,                          // HTTP/HTTPS
		`^git@`,                               // SSH
		`^ssh://`,                             // SSH
		`^git://`,                             // Git protocol
	}
	
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}
	
	return false
}

// ParseRepoURL parses a Git repository URL to extract provider info
func ParseRepoURL(repoURL string) (provider string, cleanURL string) {
	// GitHub
	if strings.Contains(repoURL, "github.com") {
		provider = "github"
		return
	}
	
	// GitLab
	if strings.Contains(repoURL, "gitlab.com") || strings.Contains(repoURL, "gitlab") {
		provider = "gitlab"
		return
	}
	
	// Bitbucket
	if strings.Contains(repoURL, "bitbucket.org") {
		provider = "bitbucket"
		return
	}
	
	provider = "other"
	cleanURL = repoURL
	return
}

// CreateDeployment creates a new deployment record
func (r *AppRepository) CreateDeployment(ctx context.Context, dep *database.Deployment) error {
	query := `
		INSERT INTO deployments (id, app_id, status, commit_sha, commit_message, commit_author, 
			branch, build_strategy, build_logs, build_duration_seconds, deploy_duration_seconds,
			triggered_by, triggered_by_user_id, rollback_of_id, started_at, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		dep.ID, dep.AppID, dep.Status, dep.CommitSHA, dep.CommitMessage, dep.CommitAuthor,
		dep.Branch, dep.BuildStrategy, dep.BuildLogs, dep.BuildDurationSeconds, dep.DeployDurationSeconds,
		dep.TriggeredBy, dep.TriggeredByUserID, dep.RollbackOfID, dep.StartedAt, dep.CompletedAt, dep.CreatedAt,
	)
	
	return err
}

// GetDeploymentByID returns a deployment by ID
func (r *AppRepository) GetDeploymentByID(ctx context.Context, id string) (*database.Deployment, error) {
	var dep database.Deployment
	err := r.db.GetContext(ctx, &dep, "SELECT * FROM deployments WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &dep, nil
}

// GetDeploymentsByAppID returns all deployments for an app
func (r *AppRepository) GetDeploymentsByAppID(ctx context.Context, appID string, status string, page, perPage int) ([]database.Deployment, int, error) {
	query := "SELECT * FROM deployments WHERE app_id = ?"
	countQuery := "SELECT COUNT(*) FROM deployments WHERE app_id = ?"
	var args []interface{}
	args = append(args, appID)
	
	if status != "" && status != "all" {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, status)
	}
	
	// Get total count
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	
	// Apply sorting and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * perPage
	args = append(args, perPage, offset)
	
	var deps []database.Deployment
	err = r.db.SelectContext(ctx, &deps, query, args...)
	if err != nil {
		return nil, 0, err
	}
	if deps == nil {
		deps = []database.Deployment{}
	}
	
	return deps, total, nil
}

// UpdateDeploymentStatus updates deployment status
func (r *AppRepository) UpdateDeploymentStatus(ctx context.Context, id string, status string) error {
	query := "UPDATE deployments SET status = ?"
	args := []interface{}{status}
	
	if status == "in_progress" {
		query += ", started_at = ?"
		args = append(args, time.Now())
	} else if status == "success" || status == "failed" || status == "cancelled" {
		query += ", completed_at = ?"
		args = append(args, time.Now())
	}
	
	query += " WHERE id = ?"
	args = append(args, id)
	
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// UpdateAppContainer updates app container information after deployment
func (r *AppRepository) UpdateAppContainer(ctx context.Context, appID, containerID, image string, port int) error {
	query := `UPDATE apps SET container_id = ?, container_image = ?, internal_port = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, containerID, image, port, time.Now(), appID)
	return err
}

// UpdateDeploymentLogs updates deployment build logs
func (r *AppRepository) UpdateDeploymentLogs(ctx context.Context, id string, logs string, durationSeconds int) error {
	_, err := r.db.ExecContext(ctx, 
		"UPDATE deployments SET build_logs = ?, build_duration_seconds = ? WHERE id = ?",
		logs, durationSeconds, id)
	return err
}

// GetUsernameByID returns a username by user ID
func (r *AppRepository) GetUsernameByID(ctx context.Context, userID int) (string, error) {
	var username string
	err := r.db.GetContext(ctx, &username, "SELECT username FROM users WHERE id = ?", userID)
	if err == sql.ErrNoRows {
		return "unknown", nil
	}
	return username, err
}

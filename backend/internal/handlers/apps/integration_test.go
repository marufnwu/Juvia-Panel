package apps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"panel-api/internal/config"
	"panel-api/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory SQLite database with the full app schema
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Connect("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	_, err = db.Exec(createSchemaSQL)
	require.NoError(t, err)

	return db
}

const createSchemaSQL = `
CREATE TABLE apps (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	status TEXT DEFAULT 'deploying',
	health_status TEXT DEFAULT 'unknown',
	runtime TEXT DEFAULT 'static',
	runtime_version TEXT,
	source_type TEXT DEFAULT 'git',
	source_config TEXT DEFAULT '{}',
	build_strategy TEXT DEFAULT 'nixpacks',
	build_config TEXT,
	container_id TEXT,
	container_image TEXT,
	internal_port INTEGER DEFAULT 3000,
	restart_policy TEXT DEFAULT 'unless-stopped',
	health_check_path TEXT DEFAULT '/health',
	health_check_interval INTEGER DEFAULT 30,
	health_check_timeout INTEGER DEFAULT 5,
	health_check_retries INTEGER DEFAULT 3,
	cpu_limit REAL,
	memory_limit_mb INTEGER,
	memory_swap_mb INTEGER,
	created_by INTEGER DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE app_domains (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	app_id TEXT NOT NULL,
	domain TEXT NOT NULL,
	is_primary BOOLEAN DEFAULT 0,
	force_https BOOLEAN DEFAULT 0,
	ssl_status TEXT DEFAULT 'pending',
	ssl_provider TEXT,
	ssl_cert_path TEXT,
	ssl_key_path TEXT,
	ssl_issued_at TIMESTAMP,
	ssl_expires_at TIMESTAMP,
	ssl_auto_renew BOOLEAN DEFAULT 1,
	dns_valid BOOLEAN,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE TABLE app_env_vars (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	app_id TEXT NOT NULL,
	key TEXT NOT NULL,
	value TEXT,
	is_secret BOOLEAN DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE TABLE app_volumes (
	id TEXT PRIMARY KEY,
	app_id TEXT NOT NULL,
	name TEXT NOT NULL,
	host_path TEXT NOT NULL,
	container_path TEXT NOT NULL,
	size_mb INTEGER DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE TABLE deployments (
	id TEXT PRIMARY KEY,
	app_id TEXT NOT NULL,
	status TEXT DEFAULT 'queued',
	commit_sha TEXT,
	commit_message TEXT,
	commit_author TEXT,
	branch TEXT,
	build_strategy TEXT,
	build_logs TEXT,
	build_duration_seconds INTEGER,
	deploy_duration_seconds INTEGER,
	triggered_by TEXT DEFAULT 'manual',
	triggered_by_user_id INTEGER,
	rollback_of_id TEXT,
	started_at TIMESTAMP,
	completed_at TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE TABLE users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	email TEXT UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT DEFAULT 'owner',
	two_factor_secret TEXT,
	two_factor_enabled BOOLEAN DEFAULT 0,
	two_factor_backup_codes TEXT,
	avatar_url TEXT,
	last_login_at TIMESTAMP,
	last_login_ip TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE service_app_links (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	app_id TEXT NOT NULL,
	service_id TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE services (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	version TEXT DEFAULT 'latest',
	status TEXT DEFAULT 'creating',
	internal_port INTEGER DEFAULT 0,
	internal_host TEXT DEFAULT 'localhost',
	container_id TEXT,
	container_image TEXT,
	credentials TEXT DEFAULT '{}',
	memory_limit_mb INTEGER,
	cpu_limit REAL,
	data_path TEXT DEFAULT '',
	data_size_mb INTEGER DEFAULT 0,
	backup_enabled BOOLEAN DEFAULT 0,
	backup_frequency TEXT DEFAULT 'daily',
	backup_time TEXT DEFAULT '02:00',
	backup_retention_days INTEGER DEFAULT 7,
	backup_destination TEXT DEFAULT 'local',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

func TestAppRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAppRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Create source config
	sourceConfig := database.SourceConfig{
		Type:      "git",
		Provider:  "github",
		RepoURL:   "https://github.com/user/my-app.git",
		Branch:    "main",
		AutoDeploy: true,
	}

	// Create app
	app := &database.App{
		ID:             "app_test001",
		Name:           "my-test-app",
		Status:         "deploying",
		HealthStatus:   "unknown",
		Runtime:        "nodejs",
		SourceType:     "git",
		SourceConfig:   sourceConfig.ToJSON(),
		BuildStrategy:  "nixpacks",
		InternalPort:   3000,
		RestartPolicy:  "unless-stopped",
		HealthCheckPath: "/health",
		HealthCheckInterval: 30,
		HealthCheckTimeout:  5,
		HealthCheckRetries:  3,
		CreatedBy:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)

	// Test AppExists
	exists, err := repo.AppExists(ctx, "my-test-app")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.AppExists(ctx, "non-existent-app")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test GetAppByID
	retrieved, err := repo.GetAppByID(ctx, "app_test001")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "my-test-app", retrieved.Name)
	assert.Equal(t, "deploying", retrieved.Status)
	assert.Equal(t, "nodejs", retrieved.Runtime)
	assert.Equal(t, 3000, retrieved.InternalPort)

	// Test source config stored correctly
	var parsedSC database.SourceConfig
	err = database.ParseJSONField(&retrieved.SourceConfig, &parsedSC)
	assert.NoError(t, err)
	assert.Equal(t, "git", parsedSC.Type)
	assert.Equal(t, "github", parsedSC.Provider)

	// Test ListApps
	apps, total, err := repo.ListApps(ctx, ListAppsParams{
		Page:    1,
		PerPage: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, apps, 1)
	assert.Equal(t, "my-test-app", apps[0].Name)

	// Test ListApps with status filter
	apps, total, err = repo.ListApps(ctx, ListAppsParams{
		Status:  "deploying",
		Page:    1,
		PerPage: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)

	apps, total, err = repo.ListApps(ctx, ListAppsParams{
		Status:  "running",
		Page:    1,
		PerPage: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, total)

	// Test ListApps with search
	apps, total, err = repo.ListApps(ctx, ListAppsParams{
		Search:  "test",
		Page:    1,
		PerPage: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)

	apps, total, err = repo.ListApps(ctx, ListAppsParams{
		Search:  "nonexistent",
		Page:    1,
		PerPage: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestAppRepository_UpdateAndDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAppRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Create app first
	app := &database.App{
		ID:             "app_test002",
		Name:           "update-test-app",
		Status:         "running",
		HealthStatus:   "healthy",
		Runtime:        "python",
		SourceType:     "git",
		SourceConfig:   `{"type":"git","provider":"github","repo_url":"https://github.com/user/repo"}`,
		BuildStrategy:  "auto",
		InternalPort:   8000,
		RestartPolicy:  "always",
		HealthCheckPath: "/health",
		HealthCheckInterval: 10,
		HealthCheckTimeout:  3,
		HealthCheckRetries:  2,
		CreatedBy:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)

	// Update app status
	err = repo.UpdateAppStatus(ctx, "app_test002", "stopped")
	require.NoError(t, err)

	retrieved, err := repo.GetAppByID(ctx, "app_test002")
	require.NoError(t, err)
	assert.Equal(t, "stopped", retrieved.Status)

	// Update app fully
	retrieved.Status = "running"
	retrieved.HealthStatus = "degraded"
	memLimit := 512
	retrieved.MemoryLimitMB = &memLimit
	err = repo.UpdateApp(ctx, retrieved)
	require.NoError(t, err)

	retrieved, err = repo.GetAppByID(ctx, "app_test002")
	require.NoError(t, err)
	assert.Equal(t, "running", retrieved.Status)
	assert.Equal(t, "degraded", retrieved.HealthStatus)
	assert.Equal(t, 512, *retrieved.MemoryLimitMB)

	// Delete app
	err = repo.DeleteApp(ctx, "app_test002")
	require.NoError(t, err)

	// Verify deleted
	retrieved, err = repo.GetAppByID(ctx, "app_test002")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)

	exists, err := repo.AppExists(ctx, "update-test-app")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestAppRepository_Domains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAppRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Create app
	app := &database.App{
		ID:         "app_domain001",
		Name:       "domain-test-app",
		Status:     "running",
		Runtime:    "static",
		SourceType: "git",
		SourceConfig: `{"type":"git"}`,
		CreatedBy:  1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)

	// Add primary domain
	domain := &database.AppDomain{
		AppID:      "app_domain001",
		Domain:     "example.com",
		IsPrimary:  true,
		ForceHTTPS: true,
		SSLStatus:  "pending",
		CreatedAt:  now,
	}
	err = repo.CreateAppDomain(ctx, domain)
	require.NoError(t, err)

	// Add secondary domain
	domain2 := &database.AppDomain{
		AppID:     "app_domain001",
		Domain:    "www.example.com",
		IsPrimary: false,
		CreatedAt: now,
	}
	err = repo.CreateAppDomain(ctx, domain2)
	require.NoError(t, err)

	// Get domains
	domains, err := repo.GetAppDomains(ctx, "app_domain001")
	assert.NoError(t, err)
	assert.Len(t, domains, 2)

	// Get detailed domains
	domsDetail, err := repo.GetAppDomainsDetail(ctx, "app_domain001")
	assert.NoError(t, err)
	assert.Len(t, domsDetail, 2)

	// Find primary
	var primaryDomain *database.AppDomain
	for _, d := range domsDetail {
		if d.IsPrimary {
			primaryDomain = &d
			break
		}
	}
	require.NotNil(t, primaryDomain)
	assert.Equal(t, "example.com", primaryDomain.Domain)
	assert.True(t, primaryDomain.ForceHTTPS)
}

func TestAppRepository_EnvVars(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAppRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Create app
	app := &database.App{
		ID:         "app_env001",
		Name:       "env-test-app",
		Status:     "running",
		Runtime:    "nodejs",
		SourceType: "git",
		SourceConfig: `{"type":"git"}`,
		CreatedBy:  1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)

	// Insert env vars
	envVars := []database.AppEnvVar{
		{AppID: "app_env001", Key: "PORT", Value: "3000", IsSecret: false, CreatedAt: now, UpdatedAt: now},
		{AppID: "app_env001", Key: "API_KEY", Value: "secret-123", IsSecret: true, CreatedAt: now, UpdatedAt: now},
		{AppID: "app_env001", Key: "NODE_ENV", Value: "production", IsSecret: false, CreatedAt: now, UpdatedAt: now},
	}

	for _, ev := range envVars {
		_, err := repo.InsertEnvVar(ctx, &ev)
		require.NoError(t, err)
	}

	// Get env vars
	vars, err := repo.GetAppEnvVars(ctx, "app_env001")
	assert.NoError(t, err)
	assert.Len(t, vars, 3)

	// Get env count
	count, err := repo.GetEnvCount(ctx, "app_env001")
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	// Check secret flags
	secretCount := 0
	for _, v := range vars {
		if v.IsSecret {
			secretCount++
			assert.Equal(t, "secret-123", v.Value)
		}
	}
	assert.Equal(t, 1, secretCount)

	// Update env var
	err = repo.UpdateEnvVar(ctx, "app_env001", "PORT", "8080", false)
	assert.NoError(t, err)

	// Verify update
	var updated database.AppEnvVar
	err = repo.GetEnvVar(ctx, "app_env001", "PORT", &updated)
	assert.NoError(t, err)
	assert.Equal(t, "8080", updated.Value)

	// Delete env var
	err = repo.DeleteEnvVar(ctx, "app_env001", "NODE_ENV")
	assert.NoError(t, err)

	vars, err = repo.GetAppEnvVars(ctx, "app_env001")
	assert.NoError(t, err)
	assert.Len(t, vars, 2)
}

func TestAppRepository_Deployments(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAppRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Create app
	app := &database.App{
		ID:         "app_dep001",
		Name:       "deploy-test-app",
		Status:     "deploying",
		Runtime:    "nodejs",
		SourceType: "git",
		SourceConfig: `{"type":"git","provider":"github","repo_url":"https://github.com/user/repo"}`,
		CreatedBy:  1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)

	// Create deployments
	startedAt := time.Now().Add(-10 * time.Minute)
	completedAt := time.Now().Add(-5 * time.Minute)
	branch := "main"
	buildLogs := "INFO: Starting build\nERROR: Build failed"

	deps := []database.Deployment{
		{
			ID: "dep_001", AppID: "app_dep001", Status: "success",
			CommitSHA: strPtr("abc123"), Branch: &branch,
			BuildDurationSeconds: intPtr(120), DeployDurationSeconds: intPtr(30),
			StartedAt: &startedAt, CompletedAt: &completedAt,
			TriggeredBy: "manual", CreatedAt: now,
		},
		{
			ID: "dep_002", AppID: "app_dep001", Status: "failed",
			CommitSHA: strPtr("def456"), Branch: &branch,
			BuildLogs: &buildLogs,
			TriggeredBy: "git_push", CreatedAt: now,
		},
		{
			ID: "dep_003", AppID: "app_dep001", Status: "queued",
			TriggeredBy: "manual", CreatedAt: now,
		},
	}

	for _, dep := range deps {
		err := repo.CreateDeployment(ctx, &dep)
		require.NoError(t, err)
	}

	// Get all deployments
	allDeps, total, err := repo.GetDeploymentsByAppID(ctx, "app_dep001", "all", 1, 20)
	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, allDeps, 3)

	// Filter by status
	failedDeps, total, err := repo.GetDeploymentsByAppID(ctx, "app_dep001", "failed", 1, 20)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, failedDeps, 1)
	assert.Equal(t, "dep_002", failedDeps[0].ID)

	// Get single deployment
	dep, err := repo.GetDeploymentByID(ctx, "dep_001")
	require.NoError(t, err)
	require.NotNil(t, dep)
	assert.Equal(t, "success", dep.Status)
	assert.Equal(t, "abc123", *dep.CommitSHA)

	// Update deployment status
	err = repo.UpdateDeploymentStatus(ctx, "dep_003", "in_progress")
	assert.NoError(t, err)

	dep, err = repo.GetDeploymentByID(ctx, "dep_003")
	require.NoError(t, err)
	assert.Equal(t, "in_progress", dep.Status)
	assert.NotNil(t, dep.StartedAt)

	// Update to success
	err = repo.UpdateDeploymentStatus(ctx, "dep_003", "success")
	assert.NoError(t, err)

	dep, err = repo.GetDeploymentByID(ctx, "dep_003")
	require.NoError(t, err)
	assert.Equal(t, "success", dep.Status)
	assert.NotNil(t, dep.CompletedAt)

	// Update build logs
	err = repo.UpdateDeploymentLogs(ctx, "dep_001", "INFO: Rebuilding\nDONE", 90)
	assert.NoError(t, err)

	dep, err = repo.GetDeploymentByID(ctx, "dep_001")
	require.NoError(t, err)
	assert.Equal(t, 90, *dep.BuildDurationSeconds)

	// Test GetUsernameByID
	username, err := repo.GetUsernameByID(ctx, 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
}

func TestAppsHandler_ListAppsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	// Insert some test apps via repo
	repo := NewAppRepository(db)
	now := time.Now()
	app1 := &database.App{
		ID: "app_list001", Name: "app-one", Status: "running", Runtime: "nodejs",
		SourceType: "git", SourceConfig: `{"type":"git"}`, CreatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	app2 := &database.App{
		ID: "app_list002", Name: "app-two", Status: "stopped", Runtime: "python",
		SourceType: "git", SourceConfig: `{"type":"git"}`, CreatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	repo.CreateApp(context.Background(), app1)
	repo.CreateApp(context.Background(), app2)

	// Create handler with nil agent and wsHub (not needed for ListApps)
	handler := NewHandler(db, &config.Config{}, nil, nil)

	// Test ListApps - all
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps?page=1&per_page=20", nil)
	c.Set("request_id", "req-001")

	handler.ListApps(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data, ok := resp["data"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, data, 2)

	meta := resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["total"])
	assert.Equal(t, float64(1), meta["total_pages"])

	// Test ListApps - filter by status
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps?status=running", nil)
	c.Set("request_id", "req-002")

	handler.ListApps(c)

	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestAppsHandler_GetAppEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAppRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Create app with full data
	sourceConfig := database.SourceConfig{
		Type:     "git",
		Provider: "github",
		RepoURL:  "https://github.com/user/app.git",
		Branch:   "main",
	}
	buildConfig := database.BuildConfig{
		Strategy:     "nixpacks",
		BuildCommand: "npm run build",
		StartCommand: "npm start",
	}
	bcJSON := buildConfig.ToJSON()

	app := &database.App{
		ID: "app_get001", Name: "get-test-app", Status: "running", HealthStatus: "healthy",
		Runtime: "nodejs", SourceType: "git", SourceConfig: sourceConfig.ToJSON(),
		BuildStrategy: "nixpacks", BuildConfig: &bcJSON,
		InternalPort: 3000, RestartPolicy: "unless-stopped",
		HealthCheckPath: "/healthz", HealthCheckInterval: 15,
		CreatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)

	// Add domains
	repo.CreateAppDomain(ctx, &database.AppDomain{
		AppID: "app_get001", Domain: "myapp.example.com", IsPrimary: true, CreatedAt: now,
	})
	repo.CreateAppDomain(ctx, &database.AppDomain{
		AppID: "app_get001", Domain: "www.myapp.example.com", IsPrimary: false, CreatedAt: now,
	})

	handler := NewHandler(db, &config.Config{}, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps/app_get001", nil)
	c.Params = gin.Params{{Key: "id", Value: "app_get001"}}
	c.Set("request_id", "req-003")

	handler.GetApp(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var detail database.AppDetail
	err = json.Unmarshal(w.Body.Bytes(), &detail)
	require.NoError(t, err)

	assert.Equal(t, "app_get001", detail.ID)
	assert.Equal(t, "get-test-app", detail.Name)
	assert.Equal(t, "running", detail.Status)
	assert.Equal(t, "healthy", detail.HealthStatus)
	assert.Equal(t, "nodejs", detail.Runtime)
	assert.Equal(t, "github", detail.Source.Provider)
	assert.Equal(t, "main", detail.Source.Branch)
	assert.Len(t, detail.Domains, 2)

	// Primary domain should be set
	require.NotNil(t, detail.PrimaryDomain)
	assert.Equal(t, "myapp.example.com", *detail.PrimaryDomain)
}

func TestAppsHandler_AppNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	handler := NewHandler(db, &config.Config{}, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
	c.Set("request_id", "req-004")

	handler.GetApp(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "app_not_found")
}

func TestAppsHandler_CreateAppEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	// Insert test user
	_, err := db.Exec("INSERT INTO users (id, username, email, password_hash, role) VALUES (1, 'admin', 'admin@test.com', 'hash', 'owner')")
	require.NoError(t, err)

	cfg := &config.Config{
		MasterKey: "this-is-a-32-byte-master-key!!",
	}
	handler := NewHandler(db, cfg, nil, nil)

	body := `{
		"name": "new-real-app",
		"source": {
			"type": "git",
			"repo_url": "https://github.com/user/node-test-app.git",
			"branch": "main",
			"auto_deploy": true
		},
		"build": {
			"strategy": "nixpacks",
			"build_command": "npm run build",
			"start_command": "npm start"
		},
		"domain": {
			"primary": "test.example.com",
			"force_https": true
		},
		"environment": {
			"NODE_ENV": "production",
			"PORT": "3000",
			"API_SECRET": "supersecret123"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/apps", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "req-create")
	c.Set("user_id", 1)

	handler.CreateApp(c)

	assert.Equal(t, http.StatusCreated, w.Code, "Response: %s", w.Body.String())

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "new-real-app", resp["name"])
	assert.Equal(t, "deploying", resp["status"])
	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["deployment_id"])

	// Verify app exists in DB
	repo := NewAppRepository(db)
	appID := resp["id"].(string)
	app, err := repo.GetAppByID(context.Background(), appID)
	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Equal(t, "new-real-app", app.Name)
	assert.Equal(t, "deploying", app.Status)
	assert.Equal(t, "nodejs", app.Runtime)

	// Verify domain was created
	domains, err := repo.GetAppDomains(context.Background(), appID)
	assert.NoError(t, err)
	assert.Len(t, domains, 1)

	// Verify env vars were created
	envVars, err := repo.GetAppEnvVars(context.Background(), appID)
	assert.NoError(t, err)
	assert.Len(t, envVars, 3)

	// Verify deployment was created
	deps, total, err := repo.GetDeploymentsByAppID(context.Background(), appID, "all", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, deps, 1)
	assert.Equal(t, "queued", deps[0].Status)
}

func TestAppsHandler_CreateApp_NameExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	// Create existing app
	repo := NewAppRepository(db)
	now := time.Now()
	repo.CreateApp(context.Background(), &database.App{
		ID: "app_existing", Name: "existing-app", Status: "running", Runtime: "static",
		SourceType: "git", SourceConfig: `{"type":"git"}`, CreatedBy: 1, CreatedAt: now, UpdatedAt: now,
	})

	handler := NewHandler(db, &config.Config{}, nil, nil)

	body := `{"name": "existing-app", "source": {"type": "git", "repo_url": "https://github.com/user/repo"}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/apps", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateApp(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "app_name_exists")
}

func TestAppsHandler_CreateApp_InvalidName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	handler := NewHandler(db, &config.Config{}, nil, nil)

	body := `{"name": "bad name!", "source": {"type": "git", "repo_url": "https://github.com/user/repo"}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/apps", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateApp(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_name")
}

func TestAppsHandler_AppLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	// Insert test user
	_, err := db.Exec("INSERT INTO users (id, username, email, password_hash, role) VALUES (1, 'admin', 'admin@test.com', 'hash', 'owner')")
	require.NoError(t, err)

	cfg := &config.Config{
		MasterKey: "this-is-a-32-byte-master-key!!",
	}
	handler := NewHandler(db, cfg, nil, nil)
	repo := NewAppRepository(db)
	ctx := context.Background()

	// Step 1: Create app
	createBody := `{
		"name": "lifecycle-app",
		"source": {"type": "git", "repo_url": "https://github.com/user/lifecycle.git", "branch": "main"},
		"environment": {"APP_ENV": "staging"}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/apps", strings.NewReader(createBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.CreateApp(c)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	appID := createResp["id"].(string)

	// Step 2: List apps and find our app
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps", nil)

	handler.ListApps(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	data := listResp["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 1)

	// Step 3: Get app detail
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps/"+appID, nil)
	c.Params = gin.Params{{Key: "id", Value: appID}}

	handler.GetApp(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var detail database.AppDetail
	json.Unmarshal(w.Body.Bytes(), &detail)
	assert.Equal(t, "lifecycle-app", detail.Name)
	assert.Equal(t, "deploying", detail.Status)

	// Step 4: List deployments for this app
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps/"+appID+"/deployments", nil)
	c.Params = gin.Params{{Key: "id", Value: appID}}

	handler.ListDeployments(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var depResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &depResp)
	depData := depResp["data"].([]interface{})
	assert.Len(t, depData, 1)

	// Step 5: Get env vars
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps/"+appID+"/env", nil)
	c.Params = gin.Params{{Key: "id", Value: appID}}

	handler.GetEnvVars(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var envResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &envResp)
	variables := envResp["variables"].([]interface{})
	assert.Len(t, variables, 1)

	// Step 6: Update app status directly via repo (simulating agent completion)
	err = repo.UpdateAppStatus(ctx, appID, "running")
	assert.NoError(t, err)

	// Step 7: Verify app status updated
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/apps/"+appID, nil)
	c.Params = gin.Params{{Key: "id", Value: appID}}

	handler.GetApp(c)
	assert.Equal(t, http.StatusOK, w.Code)

	json.Unmarshal(w.Body.Bytes(), &detail)
	assert.Equal(t, "running", detail.Status)

	// Step 8: Update env vars
	updateEnvBody := `{
		"variables": [
			{"key": "APP_ENV", "value": "production", "is_secret": false},
			{"key": "NEW_VAR", "value": "hello", "is_secret": false}
		],
		"delete_keys": []
	}`

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/apps/"+appID+"/env", strings.NewReader(updateEnvBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: appID}}

	handler.UpdateEnvVars(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify env vars updated
	envVars, err := repo.GetAppEnvVars(ctx, appID)
	assert.NoError(t, err)
	assert.Len(t, envVars, 2) // APP_ENV updated, NEW_VAR added

	// Step 9: Delete app
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/apps/"+appID+"?force=true", nil)
	c.Params = gin.Params{{Key: "id", Value: appID}}

	handler.DeleteApp(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify app deleted
	app, err := repo.GetAppByID(ctx, appID)
	assert.NoError(t, err)
	assert.Nil(t, app)
}

func intPtr(i int) *int       { return &i }

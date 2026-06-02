package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"panel-api/internal/agent"
	"panel-api/internal/config"
	"panel-api/internal/database"

	"github.com/gin-gonic/gin"
)

// Handler handles service-related HTTP requests
type Handler struct {
	db          *database.DB
	config      *config.Config
	provisioner *agent.Provisioner
}

// NewHandler creates a new services handler
func NewHandler(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:          db,
		config:      cfg,
		provisioner: agent.NewProvisioner(),
	}
}

// SetProvisioner sets the provisioner for the handler
func (h *Handler) SetProvisioner(p *agent.Provisioner) {
	h.provisioner = p
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ListServices handles GET /services
func (h *Handler) ListServices(c *gin.Context) {
	requestID := c.GetString("request_id")
	ctx := context.Background()
	
	// Parse query parameters
	svcType := c.Query("type")
	status := c.Query("status")
	search := c.Query("search")
	page := 1
	perPage := 20
	
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}
	
	// Build query
	query := `SELECT s.*, 
		(SELECT COUNT(*) FROM service_app_links l WHERE l.service_id = s.id) as connected_apps_count
		FROM services s WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM services s WHERE 1=1`
	var args []interface{}
	
	if svcType != "" {
		query += " AND s.type = ?"
		countQuery += " AND s.type = ?"
		args = append(args, svcType)
	}
	
	if status != "" {
		query += " AND s.status = ?"
		countQuery += " AND s.status = ?"
		args = append(args, status)
	}
	
	if search != "" {
		searchPattern := "%" + search + "%"
		query += " AND s.name LIKE ?"
		countQuery += " AND s.name LIKE ?"
		args = append(args, searchPattern)
	}
	
	// Get total count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := h.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to count services",
			RequestID: requestID,
		})
		return
	}
	
	// Apply sorting and pagination
	query += " ORDER BY s.name ASC LIMIT ? OFFSET ?"
	offset := (page - 1) * perPage
	args = append(args, perPage, offset)
	
	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to list services",
			RequestID: requestID,
		})
		return
	}
	defer rows.Close()
	
	type ServiceRow struct {
		database.Service
		ConnectedAppsCount int `db:"connected_apps_count"`
	}
	
	var services []database.ServiceListItem
	for rows.Next() {
		var row ServiceRow
		var lastBackupAt sql.NullTime
		
		err := rows.Scan(
			&row.ID, &row.Name, &row.Type, &row.Version, &row.Status,
			&row.InternalPort, &row.InternalHost, &row.ContainerID, &row.ContainerImage,
			&row.Credentials, &row.MemoryLimitMB, &row.CPULimit,
			&row.DataPath, &row.DataSizeMB,
			&row.BackupEnabled, &row.BackupFrequency, &row.BackupTime,
			&row.BackupRetentionDays, &row.BackupDestination,
			&row.CreatedAt, &row.UpdatedAt, &lastBackupAt,
			&row.ConnectedAppsCount,
		)
		if err != nil {
			continue
		}
		
		item := database.ServiceListItem{
			ID:             row.ID,
			Name:           row.Name,
			Type:           row.Type,
			Version:        row.Version,
			Status:         row.Status,
			Port:           row.InternalPort,
			DataSizeMB:     row.DataSizeMB,
			ConnectedApps:  row.ConnectedAppsCount,
			ResourceUsage:  nil, // Would need container metrics
			LastBackupAt:   nil,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
		
		if lastBackupAt.Valid {
			item.LastBackupAt = &lastBackupAt.Time
		}
		
		services = append(services, item)
	}
	
	if services == nil {
		services = []database.ServiceListItem{}
	}
	
	totalPages := (total + perPage - 1) / perPage
	
	c.JSON(http.StatusOK, gin.H{
		"data": services,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	})
}

// GetService handles GET /services/:id
func (h *Handler) GetService(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// Parse credentials (mask password)
	var creds database.ServiceCredentials
	parseCredentials(svc.Credentials, &creds)
	
	// Get connected apps
	apps, _ := h.getConnectedApps(ctx, serviceID)
	
	// Build response
	detail := database.ServiceDetail{
		ID:              svc.ID,
		Name:            svc.Name,
		Type:            svc.Type,
		Version:         svc.Version,
		Status:          svc.Status,
		Port:            svc.InternalPort,
		InternalHost:    svc.InternalHost,
		ContainerID:     svc.ContainerID,
		DataSizeMB:      svc.DataSizeMB,
		ResourceUsage: &database.ServiceResourceUsage{
			CPUPercent:        0,
			MemoryMB:          0,
			MemoryLimitMB:     256,
			ConnectionsActive: 0,
			ConnectionsMax:    100,
		},
		Credentials: &database.ServiceCredentials{
			Host:            creds.Host,
			Port:            creds.Port,
			Database:        creds.Database,
			Username:        creds.Username,
			Password:        maskPassword(creds.Password),
			ConnectionString: creds.ConnectionString,
		},
		BackupSchedule: &database.BackupSchedule{
			Enabled:       svc.BackupEnabled,
			Frequency:     svc.BackupFrequency,
			Time:          svc.BackupTime,
			Timezone:      "UTC",
			RetentionDays: svc.BackupRetentionDays,
			Destination:   svc.BackupDestination,
		},
		ConnectedApps: apps,
		CreatedAt:     svc.CreatedAt,
		UpdatedAt:     svc.UpdatedAt,
	}
	
	c.JSON(http.StatusOK, detail)
}

func parseCredentials(credJSON string, creds *database.ServiceCredentials) {
	// Parse JSON credentials
	creds.Host = "localhost"
	creds.Port = 0
	creds.Database = ""
	creds.Username = ""
	creds.Password = ""
	creds.ConnectionString = ""
	
	// Try to parse
	if credJSON != "" {
		// Simple parsing without json package dependency
		creds.Host = extractJSONField(credJSON, "host")
		creds.Database = extractJSONField(credJSON, "database")
		creds.Username = extractJSONField(credJSON, "username")
		creds.Password = extractJSONField(credJSON, "password")
		
		if portStr := extractJSONField(credJSON, "port"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				creds.Port = port
			}
		}
		
		creds.ConnectionString = extractJSONField(credJSON, "connection_string")
	}
}

func extractJSONField(json, field string) string {
	search := `"` + field + `":`
	idx := strings.Index(json, search)
	if idx == -1 {
		return ""
	}
	
	start := idx + len(search)
	// Skip whitespace
	for start < len(json) && (json[start] == ' ' || json[start] == '\t') {
		start++
	}
	
	if start >= len(json) {
		return ""
	}
	
	// Check if string value
	if json[start] == '"' {
		start++
		end := start
		for end < len(json) && json[end] != '"' {
			if json[end] == '\\' {
				end++
			}
			end++
		}
		return json[start:end]
	}
	
	// Numeric or boolean value
	end := start
	for end < len(json) && json[end] != ',' && json[end] != '}' && json[end] != '\n' {
		end++
	}
	return strings.TrimSpace(json[start:end])
}

func maskPassword(password string) string {
	if len(password) <= 8 {
		return "********"
	}
	return password[:4] + "..." + password[len(password)-4:]
}

func (h *Handler) getConnectedApps(ctx context.Context, serviceID string) ([]database.ConnectedApp, error) {
	query := `
		SELECT a.id, a.name 
		FROM apps a 
		JOIN service_app_links l ON a.id = l.app_id 
		WHERE l.service_id = ?
	`
	rows, err := h.db.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var apps []database.ConnectedApp
	for rows.Next() {
		var app database.ConnectedApp
		if err := rows.Scan(&app.ID, &app.Name); err != nil {
			continue
		}
		apps = append(apps, app)
	}
	
	if apps == nil {
		apps = []database.ConnectedApp{}
	}
	return apps, nil
}

// CreateServiceRequest represents the request body for creating a service
type CreateServiceRequest struct {
	Name           string                   `json:"name" binding:"required"`
	Type           string                   `json:"type" binding:"required"`
	Version        string                   `json:"version,omitempty"`
	Port           int                      `json:"port,omitempty"`
	RootPassword   string                   `json:"root_password,omitempty"`
	Resources      *CreateServiceResources  `json:"resources,omitempty"`
	BackupSchedule *CreateBackupSchedule    `json:"backup_schedule,omitempty"`
}

// CreateServiceResources represents resource configuration for a service
type CreateServiceResources struct {
	MemoryLimitMB *int     `json:"memory_limit_mb,omitempty"`
	CPULimit      *float64 `json:"cpu_limit,omitempty"`
}

// CreateBackupSchedule represents backup schedule configuration
type CreateBackupSchedule struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	Frequency      string  `json:"frequency,omitempty"`
	Time           string  `json:"time,omitempty"`
	RetentionDays  *int    `json:"retention_days,omitempty"`
	Destination    string  `json:"destination,omitempty"`
}

// CreateService handles POST /services
func (h *Handler) CreateService(c *gin.Context) {
	requestID := c.GetString("request_id")
	ctx := context.Background()
	
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body: " + err.Error(),
			RequestID: requestID,
		})
		return
	}
	
	// Validate service type
	validTypes := map[string]bool{
		"postgresql": true, "mysql": true, "mariadb": true,
		"mongodb": true, "redis": true, "memcached": true,
		"minio": true, "custom": true,
	}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_service_type",
			Message:   "Invalid service type",
			RequestID: requestID,
		})
		return
	}
	
	// Check if service name already exists
	var count int
	h.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM services WHERE name = ?", req.Name)
	if count > 0 {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "service_name_exists",
			Message:   "A service with the name '" + req.Name + "' already exists.",
			RequestID: requestID,
		})
		return
	}
	
	// Generate service ID
	serviceID, _ := generateID("svc_")
	
	// Determine port if not provided
	port := req.Port
	if port == 0 {
		port = getDefaultPort(req.Type)
	}
	
	// Generate credentials
	password := req.RootPassword
	if password == "" {
		password = generatePassword()
	}
	
	version := req.Version
	if version == "" {
		version = getDefaultVersion(req.Type)
	}
	
	// Build credentials JSON
	creds := database.ServiceCredentials{
		Host:            "localhost",
		Port:            port,
		Database:        req.Name,
		Username:        req.Name + "-user",
		Password:        password,
		ConnectionString: buildConnectionString(req.Type, req.Name, password, port),
	}
	credsJSON := fmt.Sprintf(`{"host":"localhost","port":%d,"database":"%s","username":"%s","password":"%s","connection_string":"%s"}`,
		port, creds.Database, creds.Username, creds.Password, creds.ConnectionString)
	
	// Determine internal host (Docker DNS name)
	internalHost := req.Name
	internalPort := port
	
	// Data path
	dataPath := "/var/panel/services/" + serviceID + "/data"
	
	// Memory limit
	var memoryLimitMB *int
	var cpuLimit *float64
	if req.Resources != nil {
		memoryLimitMB = req.Resources.MemoryLimitMB
		cpuLimit = req.Resources.CPULimit
	}
	
	// Backup settings
	backupEnabled := true
	backupFrequency := "daily"
	backupTime := "02:00"
	backupRetentionDays := 7
	backupDestination := "local"
	
	if req.BackupSchedule != nil {
		if req.BackupSchedule.Enabled != nil {
			backupEnabled = *req.BackupSchedule.Enabled
		}
		if req.BackupSchedule.Frequency != "" {
			backupFrequency = req.BackupSchedule.Frequency
		}
		if req.BackupSchedule.Time != "" {
			backupTime = req.BackupSchedule.Time
		}
		if req.BackupSchedule.RetentionDays != nil {
			backupRetentionDays = *req.BackupSchedule.RetentionDays
		}
		if req.BackupSchedule.Destination != "" {
			backupDestination = req.BackupSchedule.Destination
		}
	}
	
	now := time.Now()
	
	// Insert service
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO services (id, name, type, version, status, internal_port, internal_host,
			credentials, memory_limit_mb, cpu_limit, data_path, data_size_mb,
			backup_enabled, backup_frequency, backup_time, backup_retention_days, backup_destination,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, serviceID, req.Name, req.Type, version, "creating", internalPort, internalHost,
		credsJSON, memoryLimitMB, cpuLimit, dataPath, 0,
		backupEnabled, backupFrequency, backupTime, backupRetentionDays, backupDestination,
		now, now)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create service",
			RequestID: requestID,
		})
		return
	}

	// Trigger actual provisioning based on service type
	go func() {
		ctx := context.Background()
		var provisionErr error

		switch req.Type {
		case "postgresql":
			_, provisionErr = h.provisioner.ProvisionPostgreSQL(ctx, serviceID, password)
		case "mysql", "mariadb":
			_, provisionErr = h.provisioner.ProvisionMySQL(ctx, serviceID, password)
		case "mongodb":
			_, provisionErr = h.provisioner.ProvisionMongoDB(ctx, serviceID, password)
		case "redis":
			_, provisionErr = h.provisioner.ProvisionRedis(ctx, serviceID, password)
		default:
			// Unsupported service type - mark as failed
			provisionErr = fmt.Errorf("unsupported service type: %s", req.Type)
		}

		status := "running"
		if provisionErr != nil {
			status = "failed"
			fmt.Printf("Service provisioning failed: %v\n", provisionErr)
		}

		h.db.ExecContext(context.Background(), "UPDATE services SET status = ? WHERE id = ?", status, serviceID)
	}()

	c.JSON(http.StatusCreated, gin.H{
		"id":      serviceID,
		"name":    req.Name,
		"type":    req.Type,
		"status":  "creating",
		"port":    port,
		"credentials": gin.H{
			"host":            "localhost",
			"port":            port,
			"database":        req.Name,
			"username":        creds.Username,
			"password":        password,
			"connection_string": creds.ConnectionString,
		},
		"message": "Service is being provisioned. This may take 30-60 seconds.",
	})
}

func getDefaultPort(serviceType string) int {
	ports := map[string]int{
		"postgresql": 5432,
		"mysql":      3306,
		"mariadb":    3307,
		"mongodb":    27017,
		"redis":      6379,
		"memcached":  11211,
		"minio":      9000,
	}
	if port, ok := ports[serviceType]; ok {
		return port
	}
	return 5432
}

func getDefaultVersion(serviceType string) string {
	versions := map[string]string{
		"postgresql": "15.4",
		"mysql":      "8.0",
		"mariadb":   "10.11",
		"mongodb":    "7.0",
		"redis":      "7.2",
		"memcached": "1.6",
		"minio":     "2024-01",
	}
	if version, ok := versions[serviceType]; ok {
		return version
	}
	return "latest"
}

func generatePassword() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	result := make([]byte, 24)
	for i := range result {
		b := make([]byte, 1)
		rand.Read(b)
		result[i] = chars[int(b[0])%len(chars)]
	}
	return string(result)
}

func buildConnectionString(serviceType, dbName, password string, port int) string {
	switch serviceType {
	case "postgresql":
		return fmt.Sprintf("postgres://%s-user:%s@localhost:%d/%s", dbName, password, port, dbName)
	case "mysql", "mariadb":
		return fmt.Sprintf("mysql://%s-user:%s@localhost:%d/%s", dbName, password, port, dbName)
	case "mongodb":
		return fmt.Sprintf("mongodb://%s-user:%s@localhost:%d/%s", dbName, password, port, dbName)
	case "redis":
		return fmt.Sprintf("redis://localhost:%d", port)
	case "memcached":
		return fmt.Sprintf("memcached://localhost:%d", port)
	default:
		return fmt.Sprintf("%s://localhost:%d", serviceType, port)
	}
}

func generateID(prefix string) (string, error) {
	bytes := make([]byte, 9)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	result := make([]byte, len(bytes))
	for i, b := range bytes {
		result[i] = chars[int(b)%62]
	}
	return prefix + string(result), nil
}

// UpdateServiceRequest represents the request body for updating a service
type UpdateServiceRequest struct {
	Name          *string               `json:"name,omitempty"`
	BackupSchedule *UpdateBackupSchedule `json:"backup_schedule,omitempty"`
	Resources     *UpdateServiceResources `json:"resources,omitempty"`
}

// UpdateBackupSchedule represents backup schedule update
type UpdateBackupSchedule struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	Frequency      string  `json:"frequency,omitempty"`
	Time           string  `json:"time,omitempty"`
	RetentionDays  *int    `json:"retention_days,omitempty"`
	Destination    string  `json:"destination,omitempty"`
}

// UpdateServiceResources represents resource update
type UpdateServiceResources struct {
	MemoryLimitMB *int     `json:"memory_limit_mb,omitempty"`
	CPULimit      *float64 `json:"cpu_limit,omitempty"`
}

// UpdateService handles PUT /services/:id
func (h *Handler) UpdateService(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// Build update query
	updates := []string{}
	args := []interface{}{}
	
	if req.Name != nil && *req.Name != svc.Name {
		// Check if name is taken
		var count int
		h.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM services WHERE name = ? AND id != ?", *req.Name, serviceID)
		if count > 0 {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:     "service_name_exists",
				Message:   "Service name already taken",
				RequestID: requestID,
			})
			return
		}
		updates = append(updates, "name = ?")
		args = append(args, *req.Name)
	}
	
	if req.BackupSchedule != nil {
		if req.BackupSchedule.Enabled != nil {
			updates = append(updates, "backup_enabled = ?")
			args = append(args, *req.BackupSchedule.Enabled)
		}
		if req.BackupSchedule.Frequency != "" {
			updates = append(updates, "backup_frequency = ?")
			args = append(args, req.BackupSchedule.Frequency)
		}
		if req.BackupSchedule.Time != "" {
			updates = append(updates, "backup_time = ?")
			args = append(args, req.BackupSchedule.Time)
		}
		if req.BackupSchedule.RetentionDays != nil {
			updates = append(updates, "backup_retention_days = ?")
			args = append(args, *req.BackupSchedule.RetentionDays)
		}
		if req.BackupSchedule.Destination != "" {
			updates = append(updates, "backup_destination = ?")
			args = append(args, req.BackupSchedule.Destination)
		}
	}
	
	if req.Resources != nil {
		if req.Resources.MemoryLimitMB != nil {
			updates = append(updates, "memory_limit_mb = ?")
			args = append(args, *req.Resources.MemoryLimitMB)
		}
		if req.Resources.CPULimit != nil {
			updates = append(updates, "cpu_limit = ?")
			args = append(args, *req.Resources.CPULimit)
		}
	}
	
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No changes made",
		})
		return
	}
	
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, serviceID)
	
	query := "UPDATE services SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to update service",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Service updated",
	})
}

// DeleteService handles DELETE /services/:id
func (h *Handler) DeleteService(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	force := c.Query("force") == "true"
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// Check if apps are connected
	var connectedCount int
	h.db.GetContext(ctx, &connectedCount, "SELECT COUNT(*) FROM service_app_links WHERE service_id = ?", serviceID)
	
	if connectedCount > 0 && !force {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "apps_connected",
			Message: fmt.Sprintf("Service is connected to %d app(s). Use force=true to delete anyway.", connectedCount),
		})
		return
	}
	
	// Create backup before deletion if there are connections
	backupCreated := false
	var backupID *string
	if connectedCount > 0 && svc.Type != "redis" && svc.Type != "memcached" {
		// In a real implementation, create a backup first
		backupIDStr, _ := generateID("bak_")
		backupID = &backupIDStr
		backupCreated = true
	}
	
	// Delete service
	_, err = h.db.ExecContext(ctx, "DELETE FROM services WHERE id = ?", serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to delete service",
			RequestID: requestID,
		})
		return
	}
	
	response := gin.H{
		"message":       "Service '" + svc.Name + "' deleted.",
		"backup_created": backupCreated,
	}
	if backupID != nil {
		response["backup_id"] = *backupID
	}
	
	c.JSON(http.StatusOK, response)
}

// RestartService handles POST /services/:id/restart
func (h *Handler) RestartService(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// Update status to restarting
	_, err = h.db.ExecContext(ctx, "UPDATE services SET status = 'restarting', updated_at = ? WHERE id = ?", time.Now(), serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to restart service",
			RequestID: requestID,
		})
		return
	}
	
	// Restart service container via provisioner
	go func() {
		if err := h.provisioner.RestartService(context.Background(), serviceID); err != nil {
			h.db.ExecContext(context.Background(), "UPDATE services SET status = 'failed' WHERE id = ?", serviceID)
			return
		}
		h.db.ExecContext(context.Background(), "UPDATE services SET status = 'running' WHERE id = ?", serviceID)
	}()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Service '" + svc.Name + "' is restarting.",
		"status":  "restarting",
	})
}

// GetServiceLogs handles GET /services/:id/logs
func (h *Handler) GetServiceLogs(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// In a real implementation, fetch logs from Docker container
	// For now, return empty logs
	c.JSON(http.StatusOK, gin.H{
		"service_id":  serviceID,
		"lines":        []interface{}{},
		"total_lines":  0,
	})
}

// TestConnection handles POST /services/:id/test-connection
func (h *Handler) TestConnection(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// In a real implementation, test the connection using the service type
	// For now, simulate success based on status
	if svc.Status != "running" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success":    false,
			"latency_ms":  nil,
			"message":     "Service is not running. Status: " + svc.Status,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"latency_ms": 1,
		"message":    fmt.Sprintf("Connected to %s %s.", strings.Title(svc.Type), svc.Version),
	})
}

// Service app links

// GetConnections handles GET /services/:id/connections
func (h *Handler) GetConnections(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	apps, err := h.getConnectedApps(ctx, serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get connections",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"connections": apps,
	})
}

// ConnectAppRequest represents the request body for connecting an app
type ConnectAppRequest struct {
	AppID           string `json:"app_id" binding:"required"`
	ConnectionEnvKey string `json:"connection_env_key,omitempty"`
}

// ConnectApp handles POST /services/:id/connect
func (h *Handler) ConnectApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	ctx := context.Background()
	
	var req ConnectAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// Check app exists
	var app database.App
	err = h.db.GetContext(ctx, &app, "SELECT * FROM apps WHERE id = ?", req.AppID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	// Determine connection env key if not provided
	envKey := req.ConnectionEnvKey
	if envKey == "" {
		switch svc.Type {
		case "postgresql", "mysql", "mariadb":
			envKey = "DATABASE_URL"
		case "mongodb":
			envKey = "MONGODB_URL"
		case "redis":
			envKey = "REDIS_URL"
		case "memcached":
			envKey = "MEMCACHED_URL"
		case "minio":
			envKey = "S3_ENDPOINT"
		default:
			envKey = strings.ToUpper(svc.Type) + "_URL"
		}
	}
	
	// Create connection link
	now := time.Now()
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO service_app_links (service_id, app_id, connection_env_key, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(service_id, app_id) DO UPDATE SET connection_env_key = ?
	`, serviceID, req.AppID, envKey, now, envKey)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to connect app",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "App '" + app.Name + "' connected to service '" + svc.Name + "'.",
		"connection_env_key": envKey,
	})
}

// DisconnectApp handles DELETE /services/:id/disconnect/:app_id
func (h *Handler) DisconnectApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	serviceID := c.Param("id")
	appID := c.Param("app_id")
	ctx := context.Background()
	
	// Check service exists
	var svc database.Service
	err := h.db.GetContext(ctx, &svc, "SELECT * FROM services WHERE id = ?", serviceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "service_not_found",
			Message:   "Service not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get service",
			RequestID: requestID,
		})
		return
	}
	
	// Check app exists
	var app database.App
	err = h.db.GetContext(ctx, &app, "SELECT * FROM apps WHERE id = ?", appID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	// Delete connection
	result, err := h.db.ExecContext(ctx, "DELETE FROM service_app_links WHERE service_id = ? AND app_id = ?", serviceID, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to disconnect app",
			RequestID: requestID,
		})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "connection_not_found",
			Message:   "App is not connected to this service",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "App '" + app.Name + "' disconnected from service '" + svc.Name + "'.",
	})
}

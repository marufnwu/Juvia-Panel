package database

import (
	"encoding/json"
	"time"
)

// User represents a panel user.
type User struct {
	ID                 int        `db:"id" json:"id"`
	Username           string     `db:"username" json:"username"`
	Email              string     `db:"email" json:"email"`
	PasswordHash       string     `db:"password_hash" json:"-"`
	Role               string     `db:"role" json:"role"`
	TwoFactorSecret    *string    `db:"two_factor_secret" json:"-"`
	TwoFactorEnabled   bool       `db:"two_factor_enabled" json:"two_factor_enabled"`
	TwoFactorBackupCodes *string  `db:"two_factor_backup_codes" json:"-"`
	AvatarURL          *string    `db:"avatar_url" json:"avatar_url"`
	LastLoginAt        *time.Time `db:"last_login_at" json:"last_login_at"`
	LastLoginIP        *string    `db:"last_login_ip" json:"last_login_ip"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
}

// Session represents an active user session.
type Session struct {
	ID               string    `db:"id" json:"id"`
	UserID           int       `db:"user_id" json:"user_id"`
	RefreshTokenHash string    `db:"refresh_token_hash" json:"-"`
	IPAddress        *string   `db:"ip_address" json:"ip_address"`
	UserAgent        *string   `db:"user_agent" json:"user_agent"`
	ExpiresAt        time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID         string     `db:"id" json:"id"`
	UserID     int        `db:"user_id" json:"user_id"`
	Name       string     `db:"name" json:"name"`
	TokenHash  string     `db:"token_hash" json:"-"`
	Scopes     string     `db:"scopes" json:"scopes"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at"`
	ExpiresAt  *time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	RevokedAt  *time.Time `db:"revoked_at" json:"revoked_at"`
}

// App represents a deployed application.
type App struct {
	ID                  string    `db:"id" json:"id"`
	Name                string    `db:"name" json:"name"`
	Status              string    `db:"status" json:"status"`
	HealthStatus        string    `db:"health_status" json:"health_status"`
	Runtime             string    `db:"runtime" json:"runtime"`
	RuntimeVersion      *string   `db:"runtime_version" json:"runtime_version"`
	SourceType          string    `db:"source_type" json:"source_type"`
	SourceConfig        string    `db:"source_config" json:"source_config"`
	BuildStrategy       string    `db:"build_strategy" json:"build_strategy"`
	BuildConfig         *string   `db:"build_config" json:"build_config"`
	ContainerID         *string   `db:"container_id" json:"container_id"`
	ContainerImage      *string   `db:"container_image" json:"container_image"`
	InternalPort        int       `db:"internal_port" json:"internal_port"`
	RestartPolicy       string    `db:"restart_policy" json:"restart_policy"`
	HealthCheckPath     string    `db:"health_check_path" json:"health_check_path"`
	HealthCheckInterval int       `db:"health_check_interval" json:"health_check_interval"`
	HealthCheckTimeout  int       `db:"health_check_timeout" json:"health_check_timeout"`
	HealthCheckRetries  int       `db:"health_check_retries" json:"health_check_retries"`
	CPULimit            *float64  `db:"cpu_limit" json:"cpu_limit"`
	MemoryLimitMB       *int      `db:"memory_limit_mb" json:"memory_limit_mb"`
	MemorySwapMB        *int      `db:"memory_swap_mb" json:"memory_swap_mb"`
	CreatedBy           int       `db:"created_by" json:"created_by"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

// SourceConfig represents Git source configuration for an app.
type SourceConfig struct {
	Type        string `json:"type"`
	Provider    string `json:"provider,omitempty"`
	RepoURL     string `json:"repo_url,omitempty"`
	Branch      string `json:"branch,omitempty"`
	AutoDeploy  bool   `json:"auto_deploy,omitempty"`
	LastCommit  string `json:"last_commit,omitempty"`
	CommitMsg   string `json:"last_commit_message,omitempty"`
	CommitAuthor string `json:"last_commit_author,omitempty"`
	CommitTime  string `json:"last_commit_timestamp,omitempty"`
}

// BuildConfig represents build configuration for an app.
type BuildConfig struct {
	Strategy        string       `json:"strategy,omitempty"`
	BuildCommand    string       `json:"build_command,omitempty"`
	StartCommand    string       `json:"start_command,omitempty"`
	PreDeployHook   string       `json:"pre_deploy_hook,omitempty"`
	PostDeployHook  string       `json:"post_deploy_hook,omitempty"`
	DockerfilePath  string       `json:"dockerfile_path,omitempty"`
	HealthCheck     *HealthCheck `json:"health_check,omitempty"`
}

// HealthCheck represents health check configuration.
type HealthCheck struct {
	Path     string `json:"path"`
	Interval int    `json:"interval"`
	Timeout  int    `json:"timeout"`
	Retries  int    `json:"retries"`
}

// AppDomain represents a domain attached to an app.
type AppDomain struct {
	ID            int        `db:"id" json:"id"`
	AppID         string     `db:"app_id" json:"app_id"`
	Domain        string     `db:"domain" json:"domain"`
	IsPrimary     bool       `db:"is_primary" json:"is_primary"`
	ForceHTTPS    bool       `db:"force_https" json:"force_https"`
	SSLStatus     string     `db:"ssl_status" json:"ssl_status"`
	SSLProvider   *string    `db:"ssl_provider" json:"ssl_provider"`
	SSLCertPath   *string    `db:"ssl_cert_path" json:"ssl_cert_path"`
	SSLKeyPath    *string    `db:"ssl_key_path" json:"ssl_key_path"`
	SSLIssuedAt   *time.Time `db:"ssl_issued_at" json:"ssl_issued_at"`
	SSLExpiresAt  *time.Time `db:"ssl_expires_at" json:"ssl_expires_at"`
	SSLAutoRenew  bool       `db:"ssl_auto_renew" json:"ssl_auto_renew"`
	DNSValid      *bool      `db:"dns_valid" json:"dns_valid"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// AppEnvVar represents an environment variable for an app.
type AppEnvVar struct {
	ID        int       `db:"id" json:"id"`
	AppID     string    `db:"app_id" json:"app_id"`
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	IsSecret  bool      `db:"is_secret" json:"is_secret"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// AppVolume represents a persistent storage volume for an app.
type AppVolume struct {
	ID            string    `db:"id" json:"id"`
	AppID         string    `db:"app_id" json:"app_id"`
	Name          string    `db:"name" json:"name"`
	HostPath      string    `db:"host_path" json:"host_path"`
	ContainerPath string    `db:"container_path" json:"container_path"`
	SizeMB        int       `db:"size_mb" json:"size_mb"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// Deployment represents a deployment record for an app.
type Deployment struct {
	ID                   string     `db:"id" json:"id"`
	AppID                string     `db:"app_id" json:"app_id"`
	Status               string     `db:"status" json:"status"`
	CommitSHA            *string    `db:"commit_sha" json:"commit_sha"`
	CommitMessage        *string    `db:"commit_message" json:"commit_message"`
	CommitAuthor         *string    `db:"commit_author" json:"commit_author"`
	Branch               *string    `db:"branch" json:"branch"`
	BuildStrategy        *string    `db:"build_strategy" json:"build_strategy"`
	BuildLogs            *string    `db:"build_logs" json:"build_logs"`
	BuildDurationSeconds *int       `db:"build_duration_seconds" json:"build_duration_seconds"`
	DeployDurationSeconds *int      `db:"deploy_duration_seconds" json:"deploy_duration_seconds"`
	TriggeredBy          string     `db:"triggered_by" json:"triggered_by"`
	TriggeredByUserID    *int       `db:"triggered_by_user_id" json:"triggered_by_user_id"`
	RollbackOfID         *string    `db:"rollback_of_id" json:"rollback_of_id"`
	StartedAt            *time.Time `db:"started_at" json:"started_at"`
	CompletedAt          *time.Time `db:"completed_at" json:"completed_at"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
}

// Service represents a managed database, cache, or other backing service.
type Service struct {
	ID                string    `db:"id" json:"id"`
	Name               string   `db:"name" json:"name"`
	Type               string   `db:"type" json:"type"`
	Version            string   `db:"version" json:"version"`
	Status             string   `db:"status" json:"status"`
	InternalPort       int      `db:"internal_port" json:"internal_port"`
	InternalHost       string   `db:"internal_host" json:"internal_host"`
	ContainerID        *string  `db:"container_id" json:"container_id"`
	ContainerImage     *string  `db:"container_image" json:"container_image"`
	Credentials        string   `db:"credentials" json:"credentials"`
	MemoryLimitMB      *int     `db:"memory_limit_mb" json:"memory_limit_mb"`
	CPULimit           *float64 `db:"cpu_limit" json:"cpu_limit"`
	DataPath           string   `db:"data_path" json:"data_path"`
	DataSizeMB         int      `db:"data_size_mb" json:"data_size_mb"`
	BackupEnabled      bool     `db:"backup_enabled" json:"backup_enabled"`
	BackupFrequency    string   `db:"backup_frequency" json:"backup_frequency"`
	BackupTime         string   `db:"backup_time" json:"backup_time"`
	BackupRetentionDays int     `db:"backup_retention_days" json:"backup_retention_days"`
	BackupDestination  string   `db:"backup_destination" json:"backup_destination"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

// ServiceCredentials represents database/service credentials.
type ServiceCredentials struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Database        string `json:"database"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConnectionString string `json:"connection_string"`
}

// ServiceAppLink represents a connection between a service and an app.
type ServiceAppLink struct {
	ServiceID       string    `db:"service_id" json:"service_id"`
	AppID           string    `db:"app_id" json:"app_id"`
	ConnectionEnvKey *string  `db:"connection_env_key" json:"connection_env_key"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

// Backup represents a backup record.
type Backup struct {
	ID               string     `db:"id" json:"id"`
	TargetType       string     `db:"target_type" json:"target_type"`
	TargetID         string     `db:"target_id" json:"target_id"`
	TargetName       string     `db:"target_name" json:"target_name"`
	Status           string     `db:"status" json:"status"`
	SizeMB           *int       `db:"size_mb" json:"size_mb"`
	Destination      string     `db:"destination" json:"destination"`
	DestinationPath  string     `db:"destination_path" json:"destination_path"`
	Checksum         *string    `db:"checksum" json:"checksum"`
	ChecksumAlgorithm string    `db:"checksum_algorithm" json:"checksum_algorithm"`
	TriggeredBy      string     `db:"triggered_by" json:"triggered_by"`
	TriggeredByUserID *int      `db:"triggered_by_user_id" json:"triggered_by_user_id"`
	StartedAt        time.Time  `db:"started_at" json:"started_at"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at"`
}

// AppListItem represents an app in a list response.
type AppListItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	HealthStatus   string    `json:"health_status"`
	Runtime        string    `json:"runtime"`
	RuntimeVersion *string   `json:"runtime_version"`
	PrimaryDomain  *string   `json:"primary_domain"`
	Domains        []string  `json:"domains"`
	Source         SourceConfig `json:"source"`
	BuildStrategy  string    `json:"build_strategy"`
	ContainerID    *string   `json:"container_id"`
	Ports          Ports     `json:"ports"`
	EnvCount       int       `json:"env_count"`
	VolumeCount    int       `json:"volume_count"`
	ResourceUsage  *ResourceUsage `json:"resource_usage"`
	LastDeployedAt *time.Time `json:"last_deployed_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Ports represents container port mappings.
type Ports struct {
	Internal int  `json:"internal"`
	External *int `json:"external"`
}

// ResourceUsage represents resource usage statistics.
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      int     `json:"memory_mb"`
	MemoryLimitMB int     `json:"memory_limit_mb"`
}

// AppDetail represents detailed app information.
type AppDetail struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Status            string           `json:"status"`
	HealthStatus      string           `json:"health_status"`
	Runtime           string           `json:"runtime"`
	RuntimeVersion    *string          `json:"runtime_version"`
	PrimaryDomain     *string          `json:"primary_domain"`
	Domains           []AppDomainItem  `json:"domains"`
	Source            SourceConfig     `json:"source"`
	Build             BuildConfig      `json:"build"`
	Resources         Resources        `json:"resources"`
	Container         ContainerInfo    `json:"container"`
	Volumes           []VolumeItem     `json:"volumes"`
	ConnectedServices []ConnectedService `json:"connected_services"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// AppDomainItem represents a domain in app detail response.
type AppDomainItem struct {
	Domain        string     `json:"domain"`
	SSLStatus     string     `json:"ssl_status"`
	SSLExpiresAt  *time.Time `json:"ssl_expires_at"`
	ForceHTTPS    bool       `json:"force_https"`
}

// Resources represents resource limits.
type Resources struct {
	CPULimit       *float64 `json:"cpu_limit"`
	MemoryLimitMB  *int     `json:"memory_limit_mb"`
	MemorySwapMB   *int     `json:"memory_swap_mb"`
}

// ContainerInfo represents container information.
type ContainerInfo struct {
	ID            *string  `json:"id"`
	Image         *string  `json:"image"`
	Status        string   `json:"status"`
	RestartPolicy string   `json:"restart_policy"`
	Ports         []int    `json:"ports"`
	Network       string   `json:"network"`
}

// VolumeItem represents a volume in app detail.
type VolumeItem struct {
	ID            string `json:"id"`
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	SizeMB        int    `json:"size_mb"`
}

// ConnectedService represents a service connected to an app.
type ConnectedService struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// DeploymentListItem represents a deployment in list response.
type DeploymentListItem struct {
	ID                  string     `json:"id"`
	AppID               string     `json:"app_id"`
	Status              string     `json:"status"`
	Commit              *string    `json:"commit"`
	CommitMessage       *string    `json:"commit_message"`
	CommitAuthor        *string    `json:"commit_author"`
	Branch              *string    `json:"branch"`
	BuildDurationSeconds *int      `json:"build_duration_seconds"`
	DeployDurationSeconds *int     `json:"deploy_duration_seconds"`
	StartedAt           *time.Time `json:"started_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	TriggeredBy         string     `json:"triggered_by"`
	TriggeredByUser     string     `json:"triggered_by_user"`
}

// DeploymentDetail represents detailed deployment information.
type DeploymentDetail struct {
	ID                   string     `json:"id"`
	AppID                string     `json:"app_id"`
	Status               string     `json:"status"`
	Commit               *string    `json:"commit"`
	CommitMessage        *string    `json:"commit_message"`
	CommitAuthor         *string    `json:"commit_author"`
	Branch               *string    `json:"branch"`
	BuildLogsURL         string     `json:"build_logs_url"`
	BuildDurationSeconds *int       `json:"build_duration_seconds"`
	DeployDurationSeconds *int      `json:"deploy_duration_seconds"`
	StartedAt            *time.Time `json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at"`
	TriggeredBy          string     `json:"triggered_by"`
	TriggeredByUser      string     `json:"triggered_by_user"`
}

// DeploymentLogLine represents a single log line.
type DeploymentLogLine struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// ServiceListItem represents a service in list response.
type ServiceListItem struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Version        string          `json:"version"`
	Status         string          `json:"status"`
	Port           int             `json:"port"`
	DataSizeMB     int             `json:"data_size_mb"`
	ConnectedApps  int             `json:"connected_apps"`
	ResourceUsage  *ResourceUsage  `json:"resource_usage"`
	LastBackupAt   *time.Time      `json:"last_backup_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ServiceDetail represents detailed service information.
type ServiceDetail struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"`
	Version           string                 `json:"version"`
	Status            string                 `json:"status"`
	Port              int                    `json:"port"`
	InternalHost      string                 `json:"internal_host"`
	ContainerID       *string                `json:"container_id"`
	DataSizeMB        int                    `json:"data_size_mb"`
	ResourceUsage     *ServiceResourceUsage   `json:"resource_usage"`
	Credentials       *ServiceCredentials     `json:"credentials"`
	BackupSchedule    *BackupSchedule        `json:"backup_schedule"`
	ConnectedApps     []ConnectedApp         `json:"connected_apps"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// ServiceResourceUsage represents service-specific resource usage.
type ServiceResourceUsage struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryMB         int     `json:"memory_mb"`
	MemoryLimitMB    int     `json:"memory_limit_mb"`
	ConnectionsActive int    `json:"connections_active"`
	ConnectionsMax   int     `json:"connections_max"`
}

// BackupSchedule represents backup schedule configuration.
type BackupSchedule struct {
	Enabled        bool   `json:"enabled"`
	Frequency      string `json:"frequency"`
	Time           string `json:"time"`
	Timezone       string `json:"timezone"`
	RetentionDays  int    `json:"retention_days"`
	Destination    string `json:"destination"`
}

// ConnectedApp represents an app connected to a service.
type ConnectedApp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PaginatedResponse represents a paginated list response.
type PaginatedResponse struct {
	Data     interface{} `json:"data"`
	Meta     PaginationMeta `json:"meta"`
}

// PaginationMeta represents pagination metadata.
type PaginationMeta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// SuccessResponse represents a success message response.
type SuccessResponse struct {
	Message string `json:"message"`
}

// Helper to parse JSON fields
func ParseJSONField(jsonStr *string, target interface{}) error {
	if jsonStr == nil || *jsonStr == "" {
		return nil
	}
	return json.Unmarshal([]byte(*jsonStr), target)
}

func (s *SourceConfig) ToJSON() string {
	data, _ := json.Marshal(s)
	return string(data)
}

func (b *BuildConfig) ToJSON() string {
	data, _ := json.Marshal(b)
	return string(data)
}

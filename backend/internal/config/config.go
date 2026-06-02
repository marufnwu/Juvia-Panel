package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application.
type Config struct {
	// Version (set via ldflags at build time)
	Version string

	// Environment
	Env string

	// Directories
	DataDir   string
	ConfigDir string
	InstallDir string
	LogDir    string

	// Server
	APIHost string
	APIPort int

	// Database
	DBPath string

	// JWT
	JWTSecret     string
	JWTExpiry     time.Duration
	RefreshExpiry time.Duration

	// Master key for encryption (AES-256)
	MasterKey string

	// Encryption key for secrets
	EncryptionKey string

	// Panel domain
	PanelDomain string
	ServerDomain string
	Email       string

	// Agent
	AgentSocket string
	AgentTCPPort int

	// Caddy
	CaddyConfig  string
	CaddyDataDir string

	// Log level
	LogLevel string

	// Allowed CORS origins (comma-separated)
	AllowedOrigins string
}

// ConfigFile mirrors the YAML layout in /etc/panel/config.yml
type ConfigFile struct {
	App struct {
		Name       string `yaml:"name"`
		Host       string `yaml:"host"`
		Port       int    `yaml:"port"`
		Env        string `yaml:"env"`
		DataDir    string `yaml:"data_dir"`
		LogDir     string `yaml:"log_dir"`
		InstallDir string `yaml:"install_dir"`
	} `yaml:"app"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	Security struct {
		MasterKeyFile     string `yaml:"master_key_file"`
		JWTSecretFile     string `yaml:"jwt_secret_file"`
		EncryptionKeyFile string `yaml:"encryption_key_file"`
	} `yaml:"security"`
	Agent struct {
		Socket  string `yaml:"socket"`
		TCPPort int    `yaml:"tcp_port"`
	} `yaml:"agent"`
	Caddy struct {
		Config  string `yaml:"config"`
		DataDir string `yaml:"data_dir"`
	} `yaml:"caddy"`
	Server struct {
		Domain      string `yaml:"domain"`
		PanelDomain string `yaml:"panel_domain"`
		Email       string `yaml:"email"`
	} `yaml:"server"`
}

// Defaults
const (
	DefaultEnv           = "production"
	DefaultDataDir       = "/var/panel"
	DefaultConfigDir     = "/etc/panel"
	DefaultInstallDir    = "/opt/panel"
	DefaultLogDir        = "/var/panel/logs"
	DefaultAPIPort       = 9090
	DefaultDBPath        = "/var/panel/panel.db"
	DefaultJWTExpiry     = 15 * time.Minute
	DefaultRefreshExpiry = 168 * time.Hour
	DefaultAgentSocket   = "/var/run/panel/agent.sock"
	DefaultAgentTCPPort  = 9091
	DefaultCaddyConfig   = "/etc/panel/caddy/Caddyfile"
	DefaultCaddyDataDir  = "/var/panel/caddy"
	DefaultLogLevel      = "info"
)

// Load reads configuration from CLI flags, config file, and env vars.
// Precedence: defaults < env vars < config file < CLI flags.
func Load() (*Config, error) {
	return LoadWithVersion(Version)
}

// LoadWithVersion is Load with an explicit version (for testing).
func LoadWithVersion(version string) (*Config, error) {
	// Parse CLI flags
	var (
		configPath = flag.String("config", "", "Path to YAML config file")
		showVer    = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	// Start with defaults
	cfg := &Config{
		Version:       version,
		Env:           DefaultEnv,
		DataDir:       DefaultDataDir,
		ConfigDir:     DefaultConfigDir,
		InstallDir:    DefaultInstallDir,
		LogDir:        DefaultLogDir,
		APIHost:       "127.0.0.1",
		APIPort:       DefaultAPIPort,
		DBPath:        DefaultDBPath,
		JWTExpiry:     DefaultJWTExpiry,
		RefreshExpiry: DefaultRefreshExpiry,
		AgentSocket:   DefaultAgentSocket,
		AgentTCPPort:  DefaultAgentTCPPort,
		CaddyConfig:   DefaultCaddyConfig,
		CaddyDataDir:  DefaultCaddyDataDir,
		LogLevel:      DefaultLogLevel,
	}

	// Apply env var overrides
	applyEnvOverrides(cfg)

	// Apply config file overrides
	if *configPath != "" {
		if err := applyConfigFile(cfg, *configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", *configPath, err)
		}
	} else if envPath := os.Getenv("PANEL_CONFIG"); envPath != "" {
		if err := applyConfigFile(cfg, envPath); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", envPath, err)
		}
	}

	// Load key files
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("PANEL_JWT_SECRET is required (set in env or security.jwt_secret_file in config)")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("PANEL_JWT_SECRET must be at least 32 characters (got %d)", len(cfg.JWTSecret))
	}

	return cfg, nil
}

// applyEnvOverrides sets values from environment variables.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PANEL_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("PANEL_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("PANEL_CONFIG_DIR"); v != "" {
		cfg.ConfigDir = v
	}
	if v := os.Getenv("PANEL_INSTALL_DIR"); v != "" {
		cfg.InstallDir = v
	}
	if v := os.Getenv("PANEL_LOG_DIR"); v != "" {
		cfg.LogDir = v
	}
	if v := os.Getenv("PANEL_API_PORT"); v != "" {
		if p, ok := parseInt(v); ok {
			cfg.APIPort = p
		}
	}
	if v := os.Getenv("PANEL_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("PANEL_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("PANEL_JWT_EXPIRY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWTExpiry = d
		}
	}
	if v := os.Getenv("PANEL_REFRESH_EXPIRY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RefreshExpiry = d
		}
	}
	if v := os.Getenv("PANEL_MASTER_KEY"); v != "" {
		cfg.MasterKey = v
	}
	if v := os.Getenv("PANEL_ENCRYPTION_KEY"); v != "" {
		cfg.EncryptionKey = v
	}
	if v := os.Getenv("PANEL_DOMAIN"); v != "" {
		cfg.PanelDomain = v
	}
	if v := os.Getenv("PANEL_EMAIL"); v != "" {
		cfg.Email = v
	}
	if v := os.Getenv("PANEL_AGENT_SOCKET"); v != "" {
		cfg.AgentSocket = v
	}
	if v := os.Getenv("PANEL_AGENT_TCP_PORT"); v != "" {
		if p, ok := parseInt(v); ok {
			cfg.AgentTCPPort = p
		}
	}
	if v := os.Getenv("PANEL_CADDY_CONFIG"); v != "" {
		cfg.CaddyConfig = v
	}
	if v := os.Getenv("PANEL_CADDY_DATA_DIR"); v != "" {
		cfg.CaddyDataDir = v
	}
	if v := os.Getenv("PANEL_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("PANEL_ALLOWED_ORIGINS"); v != "" {
		cfg.AllowedOrigins = v
	}
}

// applyConfigFile reads YAML config and applies values, loading key files.
func applyConfigFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cf ConfigFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return err
	}

	if cf.App.Name != "" {
		cfg.Env = cf.App.Env
		cfg.APIHost = cf.App.Host
		if cf.App.Port != 0 {
			cfg.APIPort = cf.App.Port
		}
		if cf.App.DataDir != "" {
			cfg.DataDir = cf.App.DataDir
		}
		if cf.App.LogDir != "" {
			cfg.LogDir = cf.App.LogDir
		}
		if cf.App.InstallDir != "" {
			cfg.InstallDir = cf.App.InstallDir
		}
	}
	if cf.Database.Path != "" {
		cfg.DBPath = cf.Database.Path
	}
	if cf.Agent.Socket != "" {
		cfg.AgentSocket = cf.Agent.Socket
	}
	if cf.Agent.TCPPort != 0 {
		cfg.AgentTCPPort = cf.Agent.TCPPort
	}
	if cf.Caddy.Config != "" {
		cfg.CaddyConfig = cf.Caddy.Config
	}
	if cf.Caddy.DataDir != "" {
		cfg.CaddyDataDir = cf.Caddy.DataDir
	}
	if cf.Server.PanelDomain != "" {
		cfg.PanelDomain = cf.Server.PanelDomain
	}
	if cf.Server.Domain != "" {
		cfg.PanelDomain = cf.Server.Domain
	}
	if cf.Server.Email != "" {
		cfg.Email = cf.Server.Email
	}

	// Load key files - files take priority over env vars
	if cf.Security.JWTSecretFile != "" {
		if v, err := readTrimmedFile(cf.Security.JWTSecretFile); err == nil && v != "" {
			cfg.JWTSecret = v
		}
	}
	if cf.Security.MasterKeyFile != "" {
		if v, err := readTrimmedFile(cf.Security.MasterKeyFile); err == nil && v != "" {
			cfg.MasterKey = v
		}
	}
	if cf.Security.EncryptionKeyFile != "" {
		if v, err := readTrimmedFile(cf.Security.EncryptionKeyFile); err == nil && v != "" {
			cfg.EncryptionKey = v
		}
	}
	return nil
}

// readTrimmedFile reads a file and trims whitespace.
func readTrimmedFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func parseInt(s string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err == nil
}

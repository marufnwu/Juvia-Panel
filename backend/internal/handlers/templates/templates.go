package templates

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
	Runtimes    []string `json:"runtimes"`
	DockerURL   string `json:"docker_compose_url"`
	Variables   []TemplateVariable `json:"variables,omitempty"`
}

type TemplateVariable struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
}

var templates = []Template{
	{
		ID:          "wordpress",
		Name:        "WordPress",
		Description: "Open source blogging platform and CMS",
		Icon:        "Wp",
		Category:    "cms",
		Runtimes:    []string{"php", "mysql"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/wordpress/docker-compose.yml",
		Variables: []TemplateVariable{
			{Key: "DATABASE_PASSWORD", Label: "Database Password", Description: "Root password for MySQL", Default: "", Required: true},
		},
	},
	{
		ID:          "ghost",
		Name:        "Ghost",
		Description: "Professional publishing platform",
		Icon:        "Ghost",
		Category:    "cms",
		Runtimes:    []string{"nodejs", "mysql"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/ghost/docker-compose.yml",
		Variables: []TemplateVariable{
			{Key: "GHOST_URL", Label: "Ghost URL", Description: "Public URL for your Ghost instance", Default: "", Required: true},
		},
	},
	{
		ID:          "plausible",
		Name:        "Plausible Analytics",
		Description: "Lightweight, privacy-friendly web analytics",
		Icon:        "Plausible",
		Category:    "analytics",
		Runtimes:    []string{"elixir", "postgres"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/plausible/docker-compose.yml",
		Variables: []TemplateVariable{
			{Key: "ADMIN_EMAIL", Label: "Admin Email", Description: "Admin account email", Default: "", Required: true},
		},
	},
	{
		ID:          "nodejs-express",
		Name:        "Node.js Express",
		Description: "Simple Node.js Express API server",
		Icon:        "Node",
		Category:    "framework",
		Runtimes:    []string{"nodejs"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/nodejs-express/docker-compose.yml",
	},
	{
		ID:          "python-django",
		Name:        "Python Django",
		Description: "Python Django web framework",
		Icon:        "Python",
		Category:    "framework",
		Runtimes:    []string{"python"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/python-django/docker-compose.yml",
	},
	{
		ID:          "postgres",
		Name:        "PostgreSQL Database",
		Description: "PostgreSQL database service",
		Icon:        "Postgres",
		Category:    "database",
		Runtimes:    []string{"postgres"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/postgres/docker-compose.yml",
		Variables: []TemplateVariable{
			{Key: "POSTGRES_PASSWORD", Label: "PostgreSQL Password", Description: "Password for postgres user", Default: "", Required: true},
			{Key: "POSTGRES_DB", Label: "Database Name", Description: "Name of the default database", Default: "app", Required: false},
		},
	},
	{
		ID:          "redis",
		Name:        "Redis Cache",
		Description: "Redis in-memory data store",
		Icon:        "Redis",
		Category:    "cache",
		Runtimes:    []string{"redis"},
		DockerURL:   "https://raw.githubusercontent.com/your-org/panel-templates/main/redis/docker-compose.yml",
	},
}

func (h *Handler) ListTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": templates,
	})
}

func (h *Handler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	for _, t := range templates {
		if t.ID == id {
			c.JSON(http.StatusOK, t)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error":   "not_found",
		"message": "Template not found",
	})
}
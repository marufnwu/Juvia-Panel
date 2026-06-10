package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"panel-api/internal/agent"
	"panel-api/internal/config"
	"panel-api/internal/database"
	"panel-api/internal/handlers"
	"panel-api/internal/handlers/apps"
	"panel-api/internal/handlers/activity"
	"panel-api/internal/handlers/auth"
	"panel-api/internal/handlers/backups"
	"panel-api/internal/handlers/cron"
	"panel-api/internal/handlers/notifications"
	"panel-api/internal/handlers/domains"
	"panel-api/internal/handlers/firewall"
	"panel-api/internal/handlers/server"
	"panel-api/internal/handlers/deployments"
	"panel-api/internal/handlers/services"
	"panel-api/internal/handlers/settings"
	"panel-api/internal/handlers/templates"
	"panel-api/internal/handlers/users"
	"panel-api/internal/middleware"
	"panel-api/internal/proxy"
	"panel-api/internal/websocket"

	"github.com/gin-gonic/gin"
)

var (
	agentClient *agent.Client
	wsHub       *websocket.Hub
	caddyMgr    *proxy.CaddyManager
)

func main() {
	// Accept an optional "serve" subcommand for forward compatibility.
	// Without it, the binary runs as the API server (backward compatible).
	args := flag.Args()
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}

	// Re-parse flags with args if needed
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Parse(args)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations complete")

	// Initialize WebSocket hub
	wsHub = websocket.NewHub(cfg)
	go wsHub.Run()

	// Initialize agent client
	socketPath := cfg.AgentSocket
	if socketPath == "" {
		socketPath = "/var/run/panel/agent.sock"
	}
	agentClient = agent.NewClient(socketPath)

	// Pre-warm agent connection; log warning but don't crash if agent is not yet up.
	log.Printf("Connecting to agent at %s...", socketPath)
	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), 5*time.Second)
	if err := agentClient.ConnectContext(ctxConnect); err != nil {
		log.Printf("WARNING: Agent is not reachable at %s: %v", socketPath, err)
		log.Printf("WARNING: Agent-dependent features (app deployments, container management) will be unavailable until the agent starts.")
	} else {
		log.Printf("Agent connected")
	}
	cancelConnect()

	// Initialize Caddy manager
	caddy := proxy.New(cfg.CaddyConfig)
	caddyMgr = proxy.NewCaddyManager(caddy, agentClient)

	// Generate initial Caddyfile with panel UI routes and admin socket enabled
	panelEmail := cfg.Email
	if panelEmail == "" {
		panelEmail = "admin@localhost"
	}
	if err := caddy.GenerateCaddyfile([]proxy.AppRoute{}, panelEmail); err != nil {
		log.Printf("WARNING: Failed to generate initial Caddyfile: %v", err)
	} else {
		if err := caddy.ReloadCaddy(); err != nil {
			log.Printf("WARNING: Failed to reload Caddy: %v", err)
		} else {
			log.Println("Caddy reloaded with panel UI routes")
		}
	}

	// Create Gin router
	router := gin.New()

	// Apply global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.Logging())
	router.Use(middleware.CORS(cfg))
	router.Use(middleware.RequestID())

	// Health check endpoint (no auth required)
	router.GET("/health", handlers.HealthCheck)

	// WebSocket endpoint
	router.GET("/api/v1/stream", func(c *gin.Context) {
		wsHub.ServeWs(c.Writer, c.Request)
	})

	// Create handlers
	appsHandler := apps.NewHandler(db, cfg, agentClient, wsHub)
	deploymentsHandler := deployments.NewHandler(db, deployments.NewDeploymentRepositoryAdapter(db), agentClient, wsHub)
	servicesHandler := services.NewHandler(db, cfg)
	serverHandler := server.NewHandler()
	domainsHandler := domains.NewHandler(cfg, caddyMgr)
	firewallHandler := firewall.NewHandler()
	cronHandler := cron.NewHandler(cfg)
	backupsHandler := backups.NewHandler(cfg)
	settingsHandler := settings.New(db, cfg)
	activityHandler := activity.NewHandler(db, cfg)
	notificationsHandler := notifications.NewHandler(db)
	templatesHandler := templates.NewHandler()

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no auth required for login/refresh/register)
		authGroup := v1.Group("/auth")
		authGroup.Use(middleware.RateLimitAuth(cfg))
		{
			authGroup.POST("/login", func(c *gin.Context) {
				c.Set("config", cfg)
				c.Set("db", db)
				auth.Login(c)
			})
			authGroup.POST("/refresh", func(c *gin.Context) {
				c.Set("config", cfg)
				c.Set("db", db)
				auth.Refresh(c)
			})
			authGroup.POST("/logout", func(c *gin.Context) {
				c.Set("config", cfg)
				c.Set("db", db)
				auth.Logout(c)
			})
			authGroup.POST("/register", func(c *gin.Context) {
				c.Set("config", cfg)
				c.Set("db", db)
				auth.Register(c)
			})
			authGroup.GET("/status", func(c *gin.Context) {
				c.Set("db", db)
				auth.CheckUsersExists(c)
			})

			// 2FA routes (auth required)
			twoFAGroup := authGroup.Group("/2fa")
			twoFAGroup.Use(middleware.Auth(cfg))
			{
				twoFAGroup.POST("/setup", func(c *gin.Context) {
					c.Set("config", cfg)
					c.Set("db", db)
					auth.Setup2FA(c)
				})
				twoFAGroup.POST("/verify", func(c *gin.Context) {
					c.Set("config", cfg)
					c.Set("db", db)
					auth.Verify2FA(c)
				})
				twoFAGroup.POST("/disable", func(c *gin.Context) {
					c.Set("config", cfg)
					c.Set("db", db)
					auth.Disable2FA(c)
				})
			}
		}

		// User routes
		usersGroup := v1.Group("/users")
		{
			// GET /users/me - requires auth
			usersGroup.GET("/me", middleware.Auth(cfg), func(c *gin.Context) {
				c.Set("db", db)
				users.GetCurrentUser(c)
			})

			// GET /users/me/api-keys - requires auth
			usersGroup.GET("/me/api-keys", middleware.Auth(cfg), func(c *gin.Context) {
				c.Set("db", db)
				users.ListAPIKeys(c)
			})

			// POST /users/me/api-keys - requires auth
			usersGroup.POST("/me/api-keys", middleware.Auth(cfg), func(c *gin.Context) {
				c.Set("db", db)
				users.CreateAPIKey(c)
			})

			// DELETE /users/me/api-keys/:id - requires auth
			usersGroup.DELETE("/me/api-keys/:id", middleware.Auth(cfg), func(c *gin.Context) {
				c.Set("db", db)
				users.RevokeAPIKey(c)
			})

			// GET /users - requires auth
			usersGroup.GET("", middleware.Auth(cfg), func(c *gin.Context) {
				c.Set("db", db)
				users.ListUsers(c)
			})

			// POST /users/invite - requires admin or owner
			usersGroup.POST("/invite", middleware.Auth(cfg), middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				users.InviteUser(c)
			})

			// GET /users/invites - requires admin or owner
			usersGroup.GET("/invites", middleware.Auth(cfg), middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				users.GetUserInvites(c)
			})

			// DELETE /users/invites/:id - requires admin or owner
			usersGroup.DELETE("/invites/:id", middleware.Auth(cfg), middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				users.CancelInvite(c)
			})

			// PUT /users/:id/role - requires owner
			usersGroup.PUT("/:id/role", middleware.Auth(cfg), middleware.RequireRole("owner"), func(c *gin.Context) {
				c.Set("db", db)
				users.UpdateUserRole(c)
			})

			// DELETE /users/:id - requires owner
			usersGroup.DELETE("/:id", middleware.Auth(cfg), middleware.RequireRole("owner"), func(c *gin.Context) {
				c.Set("db", db)
				users.DeleteUser(c)
			})
		}

		// Invite acceptance routes (no auth required)
		v1.POST("/users/invites/:id/accept", func(c *gin.Context) {
			c.Set("db", db)
			users.AcceptInvite(c)
		})

		// Apps routes
		appsGroup := v1.Group("/apps")
		appsGroup.Use(middleware.Auth(cfg))
		{
			// List apps - all authenticated users
			appsGroup.GET("", func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.ListApps(c)
			})

			// Create app - requires admin or owner
			appsGroup.POST("", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.CreateApp(c)
			})

			// Get app - all authenticated users
			appsGroup.GET("/:id", func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.GetApp(c)
			})

			// Get app logs - developers and above
			appsGroup.GET("/:id/logs", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.GetAppLogs(c)
			})

			// Update app - requires admin or owner (developers can update env vars)
			appsGroup.PUT("/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.UpdateApp(c)
			})

			// Delete app - requires admin or owner
			appsGroup.DELETE("/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.DeleteApp(c)
			})

			// App lifecycle - developers and above
			appsGroup.POST("/:id/restart", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.RestartApp(c)
			})

			appsGroup.POST("/:id/stop", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.StopApp(c)
			})

			appsGroup.POST("/:id/start", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.StartApp(c)
			})

			// Environment variables - developers and above
			appsGroup.GET("/:id/env", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.GetEnvVars(c)
			})

			appsGroup.PUT("/:id/env", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.UpdateEnvVars(c)
			})

			appsGroup.POST("/:id/env/import", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.ImportEnvVars(c)
			})

			// Volumes - developers and above
			appsGroup.GET("/:id/volumes", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.GetVolumes(c)
			})

			appsGroup.POST("/:id/volumes", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.CreateVolume(c)
			})

			appsGroup.DELETE("/:id/volumes/:volume_id", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.DeleteVolume(c)
			})

			// Deployments - developers and above
			appsGroup.GET("/:id/deployments", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.ListDeployments(c)
			})

			appsGroup.POST("/:id/deploy", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.TriggerDeployment(c)
			})

			appsGroup.POST("/:id/rollback", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.Rollback(c)
			})

			// Add domain to an existing app
			appsGroup.POST("/:id/domains", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.AddDomain(c)
			})

			// Remove domain from an existing app
			appsGroup.DELETE("/:id/domains/:domain", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				appsHandler.RemoveDomain(c)
			})
		}

		// Deployment routes (standalone)
		deploymentsGroup := v1.Group("/deployments")
		deploymentsGroup.Use(middleware.Auth(cfg))
		{
			// GET /deployments/:id - developers and above
			deploymentsGroup.GET("/:id", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				deploymentsHandler.GetDeployment(c)
			})

			// GET /deployments/:id/logs - developers and above
			deploymentsGroup.GET("/:id/logs", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				deploymentsHandler.GetDeploymentLogs(c)
			})

			// POST /deployments/:id/cancel - developers and above
			deploymentsGroup.POST("/:id/cancel", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				deploymentsHandler.CancelDeployment(c)
			})
		}

		// Services routes
		servicesGroup := v1.Group("/services")
		servicesGroup.Use(middleware.Auth(cfg))
		{
			// List services - all authenticated users
			servicesGroup.GET("", func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.ListServices(c)
			})

			// Create service - requires admin or owner
			servicesGroup.POST("", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.CreateService(c)
			})

			// Get service - all authenticated users
			servicesGroup.GET("/:id", func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.GetService(c)
			})

			// Update service - requires admin or owner
			servicesGroup.PUT("/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.UpdateService(c)
			})

			// Delete service - requires admin or owner
			servicesGroup.DELETE("/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.DeleteService(c)
			})

			// Restart service - requires admin or owner
			servicesGroup.POST("/:id/restart", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.RestartService(c)
			})

			// Service logs - developers and above
			servicesGroup.GET("/:id/logs", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.GetServiceLogs(c)
			})

			// Test connection - developers and above
			servicesGroup.POST("/:id/test-connection", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.TestConnection(c)
			})

			// Service connections (app links) - developers and above
			servicesGroup.GET("/:id/connections", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.GetConnections(c)
			})

			servicesGroup.POST("/:id/connect", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.ConnectApp(c)
			})

			servicesGroup.DELETE("/:id/disconnect/:app_id", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				servicesHandler.DisconnectApp(c)
			})

			// Service backups - developers and above
			servicesGroup.POST("/:id/backups", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				serviceID := c.Param("id")
				c.Set("target_type", "service")
				c.Set("target_id", serviceID)
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.CreateBackup(c)
			})

			servicesGroup.GET("/:id/backups", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.ListBackups(c)
			})

			servicesGroup.POST("/:id/backups/:backupId/restore", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				backupID := c.Param("backupId")
				serviceID := c.Param("id")
				c.Set("backup_id", backupID)
				c.Set("target_id", serviceID)
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.RestoreBackup(c)
			})
		}

		// Server routes - admin and owner only
		serverGroup := v1.Group("/server")
		serverGroup.Use(middleware.Auth(cfg))
		{
			serverGroup.GET("", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.GetServerInfo(c)
			})

			serverGroup.GET("/metrics", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.GetServerMetrics(c)
			})

			serverGroup.GET("/processes", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.GetProcesses(c)
			})

			serverGroup.DELETE("/processes/:pid", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.KillProcess(c)
			})

			serverGroup.GET("/disks", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.GetDiskUsage(c)
			})

			serverGroup.GET("/network", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.GetNetworkInfo(c)
			})

			serverGroup.GET("/updates", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.GetUpdates(c)
			})

			serverGroup.POST("/updates/install", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.InstallUpdates(c)
			})

			serverGroup.POST("/reboot", middleware.RequireRole("owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				serverHandler.RebootServer(c)
			})
		}

		// Domains routes - developers and above
		domainsGroup := v1.Group("/domains")
		domainsGroup.Use(middleware.Auth(cfg))
		{
			domainsGroup.GET("", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("caddy", caddyMgr)
				domainsHandler.ListDomains(c)
			})

			domainsGroup.POST("", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("caddy", caddyMgr)
				domainsHandler.AddDomain(c)
			})

			domainsGroup.DELETE("/:domain", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("caddy", caddyMgr)
				domainsHandler.RemoveDomain(c)
			})

			domainsGroup.POST("/:domain/renew-ssl", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("caddy", caddyMgr)
				domainsHandler.RenewSSL(c)
			})

			domainsGroup.POST("/validate-dns", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("caddy", caddyMgr)
				domainsHandler.ValidateDNS(c)
			})
		}

		// Firewall routes - admin and owner only
		firewallGroup := v1.Group("/firewall")
		firewallGroup.Use(middleware.Auth(cfg))
		{
			firewallGroup.GET("", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				firewallHandler.GetFirewallStatus(c)
			})

			firewallGroup.POST("/rules", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				firewallHandler.AddRule(c)
			})

			firewallGroup.DELETE("/rules/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				firewallHandler.DeleteRule(c)
			})

			firewallGroup.POST("/toggle", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				firewallHandler.ToggleFirewall(c)
			})
		}

		// Cron routes - developers and above
		cronGroup := v1.Group("/cron")
		cronGroup.Use(middleware.Auth(cfg))
		{
			cronGroup.GET("", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.ListCronJobs(c)
			})

			cronGroup.POST("", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.CreateCronJob(c)
			})

			cronGroup.GET("/:id", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.GetCronJob(c)
			})

			cronGroup.PUT("/:id", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.UpdateCronJob(c)
			})

			cronGroup.DELETE("/:id", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.DeleteCronJob(c)
			})

			cronGroup.GET("/:id/history", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.GetExecutionHistory(c)
			})

			cronGroup.POST("/:id/toggle", middleware.RequireRole("admin", "owner", "developer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("agent", agentClient)
				cronHandler.ToggleCronJob(c)
			})
		}

// Backups routes - developers and above
		backupsGroup := v1.Group("/backups")
		backupsGroup.Use(middleware.Auth(cfg))
		{
			backupsGroup.GET("", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.ListBackups(c)
			})

			backupsGroup.POST("", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.CreateBackup(c)
			})

			backupsGroup.POST("/:id/restore", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.RestoreBackup(c)
			})

			backupsGroup.DELETE("/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				c.Set("db", db)
				c.Set("config", cfg)
				c.Set("agent", agentClient)
				backupsHandler.DeleteBackup(c)
			})

			backupsGroup.GET("/settings", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				backupsHandler.GetBackupSettings(c)
			})

			backupsGroup.PUT("/settings", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				backupsHandler.UpdateBackupSettings(c)
})
		}

		// Activity routes - all authenticated users
		activityGroup := v1.Group("/activity")
		activityGroup.Use(middleware.Auth(cfg))
		{
			activityGroup.GET("", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				c.Set("db", db)
				activityHandler.ListActivity(c)
			})
		}

		// Templates routes - all authenticated users
		templatesGroup := v1.Group("/templates")
		templatesGroup.Use(middleware.Auth(cfg))
		{
			templatesGroup.GET("", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				templatesHandler.ListTemplates(c)
			})

			templatesGroup.GET("/:id", middleware.RequireRole("admin", "owner", "developer", "viewer"), func(c *gin.Context) {
				templatesHandler.GetTemplate(c)
			})
		}

		// Notifications routes - all authenticated users
		notificationsGroup := v1.Group("/notifications")
		notificationsGroup.Use(middleware.Auth(cfg))
		{
			notificationsGroup.GET("", func(c *gin.Context) {
				c.Set("db", db)
				notificationsHandler.ListNotifications(c)
			})

			notificationsGroup.GET("/unread-count", func(c *gin.Context) {
				c.Set("db", db)
				notificationsHandler.GetUnreadCount(c)
			})

			notificationsGroup.POST("/:id/read", func(c *gin.Context) {
				c.Set("db", db)
				notificationsHandler.MarkAsRead(c)
			})

			notificationsGroup.POST("/read-all", func(c *gin.Context) {
				c.Set("db", db)
				notificationsHandler.MarkAllAsRead(c)
			})

			notificationsGroup.DELETE("/:id", func(c *gin.Context) {
				c.Set("db", db)
				notificationsHandler.DeleteNotification(c)
			})
		}

		// Settings routes - admin and owner only
		settingsGroup := v1.Group("/settings")
		settingsGroup.Use(middleware.Auth(cfg))
		{
			settingsGroup.GET("/panel", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.GetPanelSettings(c)
			})

			settingsGroup.PUT("/panel", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.UpdatePanelSettings(c)
			})

			settingsGroup.GET("/server", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.GetServerSettings(c)
			})

			settingsGroup.PUT("/server", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.UpdateServerSettings(c)
			})

			settingsGroup.GET("/notifications", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.GetNotificationSettings(c)
			})

			settingsGroup.PUT("/notifications", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.UpdateNotificationSettings(c)
			})

			settingsGroup.POST("/notifications/test/email", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.TestEmailNotification(c)
			})

			settingsGroup.POST("/notifications/test/webhook", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.TestWebhookNotification(c)
			})

			settingsGroup.POST("/export", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.ExportPanelData(c)
			})

			settingsGroup.GET("/export/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.GetExportStatus(c)
			})

			settingsGroup.GET("/export/download/:id", middleware.RequireRole("admin", "owner"), func(c *gin.Context) {
				settingsHandler.DownloadExport(c)
			})
		}
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)
		log.Printf("Starting Juvia API server on %s (version %s)", addr, cfg.Version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

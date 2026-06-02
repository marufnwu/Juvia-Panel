package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"panel-api/internal/config"

	"github.com/gorilla/websocket"
	"github.com/golang-jwt/jwt/v5"
)

// Event types
const (
	EventAppDeployStarted  = "app.deploy.started"
	EventAppDeployProgress = "app.deploy.progress"
	EventAppDeploySuccess  = "app.deploy.success"
	EventAppDeployFailed   = "app.deploy.failed"
	EventAppLogs           = "app.logs"
	EventAppStatusChanged  = "app.status_changed"
	EventServiceMetrics    = "service.metrics"
	EventServerMetrics     = "server.metrics"
	EventServerAlert       = "server.alert"
	EventNotification      = "notification"
)

// Channel types for subscriptions
const (
	ChannelApp         = "app"       // app.{app_id}
	ChannelServer      = "server"    // server.metrics, server.alert
	ChannelDeployments = "deployments" // all deployment events
	ChannelServices    = "services"  // all service events
	ChannelAll         = "all"       // everything
)

// Hub maintains active WebSocket connections and broadcasts events
type Hub struct {
	clients    map[*Client]bool
	channels   map[string]map[*Client]bool // channel -> clients
	broadcast  chan *Event
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
	cfg        *config.Config
}

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"payload,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	channels  map[string]bool
	userID    int
	username  string
	role      string
	authenticated bool
}

// AccessClaims for JWT validation
type AccessClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewHub creates a new Hub instance
func NewHub(cfg *config.Config) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		channels:  make(map[string]map[*Client]bool),
		broadcast:  make(chan *Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		cfg:        cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In development, allow all
				if cfg.Env == "development" {
					return true
				}
				// In production, check against whitelist
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				if cfg.AllowedOrigins == "" {
					// Fallback to panel domain check
					return cfg.PanelDomain != "" && strings.Contains(origin, cfg.PanelDomain)
				}
				allowedList := strings.Split(cfg.AllowedOrigins, ",")
				for _, allowed := range allowedList {
					if strings.TrimSpace(allowed) == origin {
						return true
					}
				}
				return false
			},
		},
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all channels
				for channel, clients := range h.channels {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.channels, channel)
						}
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case event := <-h.broadcast:
			h.mu.RLock()
			message, err := json.Marshal(event)
			if err != nil {
				h.mu.RUnlock()
				continue
			}

			// Broadcast to all clients (later we can filter by channel)
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToChannel sends an event to all clients subscribed to a channel
func (h *Hub) BroadcastToChannel(channel string, event *Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.channels[channel]
	if !ok {
		return
	}

	message, err := json.Marshal(event)
	if err != nil {
		return
	}

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Client buffer full, skip
		}
	}
}

// BroadcastToApp sends an event to all clients watching a specific app
func (h *Hub) BroadcastToApp(appID string, event *Event) {
	h.BroadcastToChannel("app."+appID, event)
}

// Broadcast sends an event to all connected clients
func (h *Hub) Broadcast(event *Event) {
	select {
	case h.broadcast <- event:
	default:
		log.Printf("Broadcast buffer full, dropping event: %s", event.Type)
	}
}

// Subscribe adds a client to a channel
func (h *Hub) Subscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true
	client.channels[channel] = true
}

// Unsubscribe removes a client from a channel
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channels[channel]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}
	delete(client.channels, channel)
}

// ServeWs handles WebSocket upgrade requests
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	// Try to validate JWT from query parameter first (token query param)
	tokenString := r.URL.Query().Get("token")
	
	var userID int
	var username string
	var role string
	var authenticated bool

	if tokenString != "" {
		// Validate token from query parameter
		claims, err := h.validateToken(tokenString)
		if err == nil && claims != nil {
			userID = claims.UserID
			username = claims.Username
			role = claims.Role
			authenticated = true
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
		channels:  make(map[string]bool),
		userID:    userID,
		username:  username,
		role:      role,
		authenticated: authenticated,
	}

	h.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// If authenticated via query param, immediately send auth success
	if authenticated {
		authMsg := mustMarshal(Event{
			Type:      "auth.success",
			Timestamp: time.Now().Format(time.RFC3339),
			Data: map[string]interface{}{
				"user_id":  userID,
				"username": username,
				"role":     role,
			},
		})
		client.send <- authMsg
	}
}

// validateToken validates a JWT token and returns the claims
func (h *Hub) validateToken(tokenString string) (*AccessClaims, error) {
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(h.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetChannelCount returns the number of active channels
func (h *Hub) GetChannelCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels)
}

// writePump pumps messages to the client connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the client connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB limit
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse incoming message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle message types
		msgType, _ := msg["type"].(string)
		switch msgType {
		case "auth":
			// Authenticate the client
			if token, ok := msg["token"].(string); ok {
				c.authenticate(token)
			}

		case "subscribe":
			// Check if authenticated first
			if !c.authenticated {
				c.send <- mustMarshal(Event{
					Type:      "error",
					Timestamp: time.Now().Format(time.RFC3339),
					Data: map[string]string{
						"code":    "unauthorized",
						"message": "Not authenticated",
					},
				})
				continue
			}
			// Subscribe to channels
			if channels, ok := msg["channels"].([]interface{}); ok {
				for _, ch := range channels {
					if channel, ok := ch.(string); ok {
						c.hub.Subscribe(c, channel)
					}
				}
			}

		case "unsubscribe":
			// Unsubscribe from channels
			if channels, ok := msg["channels"].([]interface{}); ok {
				for _, ch := range channels {
					if channel, ok := ch.(string); ok {
						c.hub.Unsubscribe(c, channel)
					}
				}
			}

		case "ping":
			// Respond to ping
			c.send <- mustMarshal(Event{Type: "pong", Timestamp: time.Now().Format(time.RFC3339)})
		}
	}
}

// authenticate authenticates the client with a JWT token
func (c *Client) authenticate(token string) {
	// Validate JWT token
	claims, err := c.hub.validateToken(token)
	if err != nil {
		log.Printf("WebSocket auth failed: %v", err)
		c.send <- mustMarshal(Event{
			Type:      "auth.error",
			Timestamp: time.Now().Format(time.RFC3339),
			Data: map[string]string{
				"code":    "invalid_token",
				"message": "Invalid or expired token",
			},
		})
		return
	}

	// Mark as authenticated
	c.authenticated = true
	c.userID = claims.UserID
	c.username = claims.Username
	c.role = claims.Role

	log.Printf("WebSocket client authenticated: user_id=%d, username=%s", claims.UserID, claims.Username)

	// Send auth success
	c.send <- mustMarshal(Event{
		Type:      "auth.success",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"role":     claims.Role,
		},
	})
}

// Helper to marshal JSON
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}
	return data
}

// EmitAppDeployStarted emits an app deploy started event
func EmitAppDeployStarted(hub *Hub, appID, deploymentID, commit, branch string) {
	hub.Broadcast(&Event{
		Type:      EventAppDeployStarted,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]string{
			"app_id":        appID,
			"deployment_id": deploymentID,
			"commit":        commit,
			"branch":        branch,
		},
	})
	hub.BroadcastToApp(appID, &Event{
		Type:      EventAppDeployStarted,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]string{
			"app_id":        appID,
			"deployment_id": deploymentID,
			"commit":        commit,
			"branch":        branch,
		},
	})
}

// EmitAppDeployProgress emits an app deploy progress event
func EmitAppDeployProgress(hub *Hub, appID, deploymentID, step, message string, percent int) {
	event := &Event{
		Type:      EventAppDeployProgress,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"app_id":        appID,
			"deployment_id": deploymentID,
			"step":          step,
			"message":       message,
			"percent":       percent,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToApp(appID, event)
}

// EmitAppDeploySuccess emits an app deploy success event
func EmitAppDeploySuccess(hub *Hub, appID, deploymentID string, durationSeconds int) {
	event := &Event{
		Type:      EventAppDeploySuccess,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"app_id":           appID,
			"deployment_id":    deploymentID,
			"duration_seconds": durationSeconds,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToApp(appID, event)
}

// EmitAppDeployFailed emits an app deploy failed event
func EmitAppDeployFailed(hub *Hub, appID, deploymentID, errorMsg, step string) {
	event := &Event{
		Type:      EventAppDeployFailed,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]string{
			"app_id":        appID,
			"deployment_id": deploymentID,
			"error":         errorMsg,
			"step":          step,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToApp(appID, event)
}

// EmitAppLogs emits an app log event
func EmitAppLogs(hub *Hub, appID, stream, message string) {
	event := &Event{
		Type:      EventAppLogs,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]string{
			"app_id":  appID,
			"stream":  stream,
			"message": message,
		},
	}
	hub.BroadcastToApp(appID, event)
}

// EmitAppStatusChanged emits an app status changed event
func EmitAppStatusChanged(hub *Hub, appID, oldStatus, newStatus, healthStatus string) {
	event := &Event{
		Type:      EventAppStatusChanged,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]string{
			"app_id":        appID,
			"old_status":    oldStatus,
			"new_status":    newStatus,
			"health_status": healthStatus,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToApp(appID, event)
}

// EmitServiceMetrics emits a service metrics event
func EmitServiceMetrics(hub *Hub, serviceID string, cpuPercent float64, memoryMB int, connections int) {
	event := &Event{
		Type:      EventServiceMetrics,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"service_id":   serviceID,
			"cpu_percent":  cpuPercent,
			"memory_mb":    memoryMB,
			"connections":  connections,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToChannel("services", event)
}

// EmitServerMetrics emits a server metrics event
func EmitServerMetrics(hub *Hub, cpuPercent, memoryPercent, diskPercent float64, load1Min float64) {
	event := &Event{
		Type:      EventServerMetrics,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"cpu_percent":    cpuPercent,
			"memory_percent": memoryPercent,
			"disk_percent":   diskPercent,
			"load_1min":      load1Min,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToChannel("server", event)
}

// EmitServerAlert emits a server alert event
func EmitServerAlert(hub *Hub, severity, metric string, value, threshold float64, message string) {
	event := &Event{
		Type:      EventServerAlert,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"severity":  severity,
			"metric":    metric,
			"value":     value,
			"threshold": threshold,
			"message":   message,
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToChannel("server", event)
}

// EmitNotification emits a notification event
func EmitNotification(hub *Hub, id, title, message, severity, link string) {
	event := &Event{
		Type:      EventNotification,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]string{
			"id":       id,
			"title":    title,
			"message":  message,
			"severity": severity,
			"link":     link,
			"read":     "false",
		},
	}
	hub.Broadcast(event)
	hub.BroadcastToChannel("all", event)
}

// Helpers for extracting token from various sources

// extractTokenFromQuery extracts token from URL query parameter
func extractTokenFromQuery(r *http.Request) string {
	return r.URL.Query().Get("token")
}

// extractTokenFromAuthHeader extracts token from Authorization header
func extractTokenFromAuthHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	
	return parts[1]
}
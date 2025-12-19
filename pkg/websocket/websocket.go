package websocket

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// UpgraderConfig holds WebSocket upgrader configuration
type UpgraderConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
	CheckOrigin     func(r *http.Request) bool
}

// DefaultUpgraderConfig returns default upgrader configuration
// Note: In production, you should configure CheckOrigin to validate origins
func DefaultUpgraderConfig() UpgraderConfig {
	return UpgraderConfig{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     defaultCheckOrigin,
	}
}

// defaultCheckOrigin checks if the origin matches the request host
// This provides basic CSRF protection while allowing same-origin connections
func defaultCheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // No origin header, likely same-origin
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	return u.Host == r.Host
}

// AllowAllOrigins is a CheckOrigin function that allows all origins
// WARNING: Only use this in development or when you have other CSRF protections
func AllowAllOrigins(r *http.Request) bool {
	return true
}

// Connection wraps websocket connection
type Connection struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
	id   string
}

// Hub maintains active connections
type Hub struct {
	connections map[*Connection]bool
	broadcast   chan []byte
	register    chan *Connection
	unregister  chan *Connection
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

// NewHub creates a new Hub with default configuration
func NewHub() *Hub {
	return NewHubWithConfig(DefaultUpgraderConfig())
}

// NewHubWithConfig creates a new Hub with custom configuration
func NewHubWithConfig(config UpgraderConfig) *Hub {
	return &Hub{
		connections: make(map[*Connection]bool),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.connections[conn] = true
			h.mu.Unlock()
			logrus.Infof("WebSocket client connected: %s", conn.id)

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				close(conn.send)
				logrus.Infof("WebSocket client disconnected: %s", conn.id)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.connections {
				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(h.connections, conn)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends message to all connections
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// ConnectionCount returns number of active connections
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// Upgrade upgrades HTTP connection to WebSocket
func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request, id string) error {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	connection := &Connection{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h,
		id:   id,
	}

	h.register <- connection

	go connection.writePump()
	go connection.readPump()

	return nil
}

func (c *Connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Broadcast received message to all clients
		c.hub.broadcast <- message
	}
}

func (c *Connection) writePump() {
	defer func() {
		_ = c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			logrus.Errorf("WebSocket write error: %v", err)
			break
		}
	}
}

// Send sends message to this specific connection
func (c *Connection) Send(message []byte) {
	c.send <- message
}

// Close closes the connection
func (c *Connection) Close() error {
	return c.conn.Close()
}

package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"xnfz/api/websocket"
	"xnfz/internal/course"
	"xnfz/internal/session"
	"xnfz/pkg/models"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Note: You may want to implement a more secure origin check
	},
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	session *session.Session
	user    *models.User
	isMain  bool
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	sessions   *session.Manager
	courses    *course.Manager
	logger     *zap.Logger
}

func NewHub(sessions *session.Manager, courses *course.Manager, logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		sessions:   sessions,
		courses:    courses,
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if client.session != nil {
					h.sessions.EndSession(client.session.ID)
				}
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
					if client.session != nil {
						h.sessions.EndSession(client.session.ID)
					}
				}
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("Unexpected close error", zap.Error(err))
			}
			break
		}

		var msg protocol.Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			c.hub.logger.Error("Error unmarshalling message", zap.Error(err))
			continue
		}

		switch msg.Type {
		case protocol.HeartbeatMessage:
			c.handleHeartbeat()
		case protocol.CourseSelectionMessage:
			c.handleCourseSelection(msg.Data)
		case protocol.ObjectManipulationMessage:
			c.hub.broadcast <- message
		case protocol.ExitCourseMessage:
			c.handleExitCourse()
		}
	}
}

func authenticateUser(r *http.Request) *models.User {
	// TODO: Implement proper user authentication
	// This is a placeholder implementation
	return &models.User{
		ID:   "user-123",
		Name: "John Doe",
		Role: models.Teacher,
	}
}

func (c *Client) handleHeartbeat() {
	response := protocol.Message{
		Type: protocol.HeartbeatResponseMessage,
		Data: "pong",
	}
	c.send <- marshalMessage(response)
}

func (c *Client) handleCourseSelection(data interface{}) {
	courseData, ok := data.(map[string]interface{})
	if !ok {
		c.hub.logger.Error("Invalid course selection data")
		return
	}

	courseID, ok := courseData["id"].(string)
	if !ok {
		c.hub.logger.Error("Invalid course ID")
		return
	}

	course, found := c.hub.courses.GetCourse(courseID)
	if !found {
		c.hub.logger.Error("Course not found", zap.String("courseID", courseID))
		return
	}

	if c.session != nil {
		c.hub.sessions.EndSession(c.session.ID)
	}

	c.session = c.hub.sessions.CreateSession(c.user, course)

	if course.Mode == models.PracticeMode {
		go c.startPracticeTimer(course.Duration)
	}

	response := protocol.Message{
		Type: protocol.CourseStartMessage,
		Data: map[string]interface{}{
			"courseID":   course.ID,
			"courseName": course.Name,
			"mode":       course.Mode,
		},
	}
	c.hub.broadcast <- marshalMessage(response)
}

func (c *Client) handleExitCourse() {
	if c.session != nil {
		c.hub.sessions.EndSession(c.session.ID)
		c.session = nil
	}

	response := protocol.Message{
		Type: protocol.CourseExitMessage,
		Data: "Course exited",
	}
	c.hub.broadcast <- marshalMessage(response)
}

func (c *Client) startPracticeTimer(duration time.Duration) {
	timer := time.NewTimer(duration)
	<-timer.C

	if c.session != nil {
		c.hub.sessions.EndSession(c.session.ID)
		c.session = nil

		response := protocol.Message{
			Type: protocol.PracticeTimeUpMessage,
			Data: "Practice time is up",
		}
		c.hub.broadcast <- marshalMessage(response)
	}
}

func marshalMessage(msg protocol.Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.logger.Error("Error upgrading connection", zap.Error(err))
		return
	}

	// Authenticate user and get user details (implement your own authentication logic)
	user := authenticateUser(r)
	if user == nil {
		hub.logger.Error("User authentication failed")
		conn.Close()
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		user:   user,
		isMain: r.URL.Query().Get("main") == "true",
	}
	client.hub.register <- client

	go client.readPump()
	go client.writePump()
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

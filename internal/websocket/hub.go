package websocket

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"xnfz/api/websocket"
	"xnfz/internal/course"
	e "xnfz/internal/errors"
	"xnfz/internal/session"
	"xnfz/pkg/models"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// 常量定义
const (
	writeWait      = 10 * time.Second    // 写操作超时时间
	pongWait       = 60 * time.Second    // 等待 pong 消息的最大时间
	pingPeriod     = (pongWait * 9) / 10 // 发送 ping 消息的周期
	maxMessageSize = 512                 // 最大消息大小
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// 更新 Message 结构体以包含 code 字段
type Message struct {
	Type string      `json:"type"`
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

// 定义状态码常量
const (
	CodeSuccess = 0
)

// WebSocket 连接升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 注意：在生产环境中应该实现更安全的源检查
	},
}

// CourseDetail 存储课程详情
type CourseDetail struct {
	CourseID       string                `json:"courseId"`
	Mode           int32                 `json:"mode"`
	DataAttachment map[int32]interface{} `json:"dataAttachment"`
	mu             sync.RWMutex
}

// GetDataAttachment 安全地获取 DataAttachment
func (cd *CourseDetail) GetDataAttachment() map[int32]interface{} {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	result := make(map[int32]interface{})
	for k, v := range cd.DataAttachment {
		result[k] = v
	}
	return result
}

// UpdateDataAttachment 安全地更新 DataAttachment
func (cd *CourseDetail) UpdateDataAttachment(update func(map[int32]interface{})) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	update(cd.DataAttachment)
}

// Client 表示一个 WebSocket 客户端连接
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	session *session.Session
	user    *models.User
	isMain  bool
}

// Hub 维护活动客户端的集合并广播消息
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	sessions     *session.Manager
	courses      *course.Manager
	logger       *zap.Logger
	courseDetail *CourseDetail
}

// NewHub 创建一个新的 Hub
func NewHub(sessions *session.Manager, courses *course.Manager, logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		sessions:   sessions,
		courses:    courses,
		logger:     logger,
		courseDetail: &CourseDetail{
			DataAttachment: make(map[int32]interface{}),
		},
	}
}

// Run 启动 Hub 的主循环
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

// readPump 从 WebSocket 连接中泵取消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
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
		case protocol.CourseModeSelectionMessage:
			c.handleCourseModeSelection(msg.Data)
		case protocol.ObjectManipulationMessage:
			c.handleObjectManipulation(msg.Data)
		case protocol.CourseExitMessage:
			c.handleExitCourse()
		}
	}
}

// authenticateUser 验证用户身份（占位实现）
func authenticateUser(r *http.Request) *models.User {
	// TODO: 实现适当的用户认证
	deviceCode := r.URL.Query().Get("deviceCode")
	role := models.Student
	if r.URL.Query().Get("main") == "true" {
		role = models.Teacher
	}
	return &models.User{
		ID:   deviceCode,
		Name: deviceCode,
		Role: role,
	}
}

// handleHeartbeat 处理心跳消息
func (c *Client) handleHeartbeat() {
	response := protocol.Message{
		Type: protocol.HeartbeatResponseMessage,
		Data: "pong",
	}
	c.send <- marshalMessage(response)
}

// handleCourseSelection 处理课程选择消息
func (c *Client) handleCourseSelection(data interface{}) {
	courseData, ok := data.(map[string]interface{})
	if !ok {
		c.hub.logger.Error("Invalid course selection data")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	courseID, ok := courseData["courseId"].(string)
	if !ok {
		c.hub.logger.Error("Invalid course ID")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	course := c.hub.courses.CreateCourse(courseID, "", "", 0, time.Duration(time.Now().Second()))

	if c.session != nil {
		c.hub.sessions.EndSession(c.session.ID)
	}

	c.session = c.hub.sessions.CreateSession(c.user, course)

	c.hub.courseDetail.CourseID = courseID

	response := protocol.Message{
		Type: protocol.CourseSelectedMessage,
		Data: map[string]interface{}{
			"courseId": courseID,
		},
	}
	c.hub.broadcast <- marshalMessage(response)
}

// handleCourseModeSelection 处理课程模式选择消息
func (c *Client) handleCourseModeSelection(data interface{}) {
	courseData, ok := data.(map[string]interface{})
	if !ok {
		c.hub.logger.Error("Invalid course selection data")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	courseID, ok := courseData["courseId"].(string)
	if !ok {
		c.hub.logger.Error("Invalid course ID")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	mode, ok := courseData["mode"].(float64)
	if !ok {
		c.hub.logger.Error("Invalid course mode")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	course, found := c.hub.courses.GetCourse(courseID)
	if !found {
		c.hub.logger.Error("Course not found", zap.String("courseID", courseID))
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrCourseNotFound.Code,
			Data: e.ErrCourseNotFound.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	c.hub.courses.UpdateCourse(courseID, "", "", models.CourseMode(mode), time.Duration(time.Now().Second()))

	if c.session != nil {
		c.hub.sessions.EndSession(c.session.ID)
	}

	c.session = c.hub.sessions.CreateSession(c.user, course)

	c.hub.courseDetail.CourseID = courseID
	c.hub.courseDetail.Mode = int32(mode)

	response := protocol.Message{
		Type: protocol.CourseStartMessage,
		Data: map[string]interface{}{
			"courseId": courseID,
			"mode":     course.Mode,
		},
	}
	c.hub.broadcast <- marshalMessage(response)
}

// handleObjectManipulation 处理对象操作消息
func (c *Client) handleObjectManipulation(data interface{}) {
	manipulationData, ok := data.(map[string]interface{})
	if !ok {
		c.hub.logger.Error("Invalid object manipulation data")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	dataAttachment, ok := manipulationData["dataAttachment"].(map[string]interface{})
	if !ok {
		c.hub.logger.Error("Invalid dataAttachment")
		response := protocol.Message{
			Type: protocol.ErrorMessage,
			Code: e.ErrInvalidData.Code,
			Data: e.ErrInvalidData.Message,
		}
		c.send <- marshalMessage(response)
		return
	}

	c.hub.courseDetail.UpdateDataAttachment(func(currentDataAttachment map[int32]interface{}) {
		for k, v := range dataAttachment {
			key, err := strconv.ParseInt(k, 10, 32)
			if err != nil {
				c.hub.logger.Error("Invalid key in dataAttachment", zap.Error(err))
				continue
			}
			currentDataAttachment[int32(key)] = v
		}
	})

	response := protocol.Message{
		Type: protocol.ObjectManipulationMessage,
		Data: manipulationData,
	}
	c.hub.broadcast <- marshalMessage(response)
}

// handleExitCourse 处理退出课程消息
func (c *Client) handleExitCourse() {
	if c.session != nil {
		c.hub.sessions.EndSession(c.session.ID)
		c.session = nil
	}

	c.hub.courseDetail.CourseID = ""
	c.hub.courseDetail.Mode = 0
	c.hub.courseDetail.UpdateDataAttachment(func(dataAttachment map[int32]interface{}) {
		for k := range dataAttachment {
			delete(dataAttachment, k)
		}
	})

	response := protocol.Message{
		Type: protocol.CourseExitMessage,
		Data: "Course exited",
	}
	c.hub.broadcast <- marshalMessage(response)

	c.hub.logger.Info("Course exited and courseDetail cleared",
		zap.String("user", c.user.ID),
		zap.String("deviceCode", c.user.ID))
}

// // startPracticeTimer 启动实践模式计时器
// func (c *Client) startPracticeTimer(duration time.Duration) {
// 	timer := time.NewTimer(duration)
// 	<-timer.C

// 	if c.session != nil {
// 		c.hub.sessions.EndSession(c.session.ID)
// 		c.session = nil

// 		response := protocol.Message{
// 			Type: protocol.PracticeTimeUpMessage,
// 			Data: "Practice time is up",
// 		}
// 		c.hub.broadcast <- marshalMessage(response)
// 	}
// }

// marshalMessage 将消息序列化为 JSON
func marshalMessage(msg protocol.Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

// ServeWs 处理 WebSocket 连接请求
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.logger.Error("Error upgrading connection", zap.Error(err))
		return
	}
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

	if hub.courseDetail.CourseID != "" {
		detailMessage := protocol.Message{
			Type: protocol.CourseDetailMessage,
			Data: hub.courseDetail,
		}
		client.send <- marshalMessage(detailMessage)
	}

	go client.readPump()
	go client.writePump()
}

// writePump 将消息泵送到 WebSocket 连接
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

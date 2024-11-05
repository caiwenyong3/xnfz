package session

import (
	"sync"
	"time"

	"xnfz/pkg/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Session struct {
	ID        string
	User      *models.User
	Course    *models.Course
	StartTime time.Time
	EndTime   time.Time
}

type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *zap.Logger
}

func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		logger:   logger,
	}
}

func (m *Manager) CreateSession(user *models.User, course *models.Course) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String(),
		User:      user,
		Course:    course,
		StartTime: time.Now(),
	}

	m.sessions[session.ID] = session
	m.logger.Info("Created new session", zap.String("sessionID", session.ID), zap.String("userID", user.ID), zap.String("courseID", course.ID))

	return session
}

func (m *Manager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	return session, ok
}

func (m *Manager) EndSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		session.EndTime = time.Now()
		m.logger.Info("Ended session", zap.String("sessionID", sessionID), zap.String("userID", session.User.ID), zap.String("courseID", session.Course.ID))
		delete(m.sessions, sessionID)
	}
}

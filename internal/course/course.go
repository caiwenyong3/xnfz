package course

import (
	"sync"
	"time"

	"xnfz/pkg/models"

	// "github.com/google/uuid"
	"go.uber.org/zap"
)

type Manager struct {
	courses map[string]*models.Course
	mu      sync.RWMutex
	logger  *zap.Logger
}

func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		courses: make(map[string]*models.Course),
		logger:  logger,
	}
}

func (m *Manager) CreateCourse(id string, name string, description string, mode models.CourseMode, duration time.Duration) *models.Course {
	m.mu.Lock()
	defer m.mu.Unlock()

	course := &models.Course{
		ID:          id,
		Name:        name,
		Description: description,
		Mode:        mode,
		Duration:    duration,
	}

	m.courses[course.ID] = course
	m.logger.Info("Created new course", zap.String("courseID", course.ID), zap.String("name", course.Name))

	return course
}

func (m *Manager) GetCourse(courseID string) (*models.Course, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	course, ok := m.courses[courseID]
	return course, ok
}

func (m *Manager) UpdateCourse(id string, name string, description string, mode models.CourseMode, duration time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if course, ok := m.courses[id]; ok {
		course.Name = name
		course.Description = description
		course.Mode = mode
		course.Duration = duration
		m.logger.Info("Updated course", zap.String("courseID", id), zap.Int16("mode", int16(mode)))
		return true
	}

	return false
}

func (m *Manager) DeleteCourse(courseID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.courses[courseID]; ok {
		delete(m.courses, courseID)
		m.logger.Info("Deleted course", zap.String("courseID", courseID))
		return true
	}

	return false
}

func (m *Manager) ListCourses() []*models.Course {
	m.mu.RLock()
	defer m.mu.RUnlock()

	courses := make([]*models.Course, 0, len(m.courses))
	for _, course := range m.courses {
		courses = append(courses, course)
	}

	return courses
}

package app

import (
	"xnfz/internal/course"
	"xnfz/internal/session"

	"go.uber.org/zap"
)

type Server struct {
	CourseManager  *course.Manager
	SessionManager *session.Manager
	Logger         *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
	return &Server{
		CourseManager:  course.NewManager(logger),
		SessionManager: session.NewManager(logger),
		Logger:         logger,
	}
}

package main

import (
	"net/http"

	"xnfz/internal/course"
	"xnfz/internal/session"
	"xnfz/internal/websocket"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	sessionManager := session.NewManager(logger)
	courseManager := course.NewManager(logger)

	hub := websocket.NewHub(sessionManager, courseManager, logger)
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	})

	logger.Info("Server is running on :9090")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		logger.Fatal("ListenAndServe: ", zap.Error(err))
	}
}

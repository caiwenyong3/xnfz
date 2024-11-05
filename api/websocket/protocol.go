package protocol

const (
	HeartbeatMessage          = "heartbeat"
	HeartbeatResponseMessage  = "heartbeat_response"
	CourseSelectionMessage    = "course_selection"
	ObjectManipulationMessage = "object_manipulation"
	CourseStartMessage        = "course_start"
	CourseExitMessage         = "course_exit"
	PracticeTimeUpMessage     = "practice_time_up"
	ExitCourseMessage         = "exit_course"
)

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

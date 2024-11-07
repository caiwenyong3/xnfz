package protocol

const (
	HeartbeatMessage         = "heartbeat"          // 心跳请求
	HeartbeatResponseMessage = "heartbeat_response" // 心跳响应
	ErrorMessage             = "error"              // 错误响应

	CourseDetailMessage    = "course_detail"    // 课程明细进度响应
	CourseSelectionMessage = "course_selection" // 课程选择请求
	CourseSelectedMessage  = "course_selected"  // 课程选择响应

	CourseModeSelectionMessage = "course_mode_selection" // 课程模式选择请求
	ObjectManipulationMessage  = "object_manipulation"   // 课程对象操作进度同步
	CourseStartMessage         = "course_start"          // 课程开始（课程模式选择响应）
	CourseExitMessage          = "course_exit"           // 课程结束
	PracticeTimeUpMessage      = "practice_time_up"
)

type Message struct {
	Type string      `json:"type"`
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

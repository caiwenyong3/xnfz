// file: internal/errors/messages.go

package errors

type ErrorMessage struct {
	Code    int16
	Message string
}

var (
	ErrInvalidData    = ErrorMessage{Code: 10001, Message: "Invalid data"}
	ErrCourseNotFound = ErrorMessage{Code: 10002, Message: "Course not found"}
	ErrInternalServer = ErrorMessage{Code: 10003, Message: "Internal server error"}
	// 添加更多错误消息...
)

// GetErrorMessage 返回给定错误代码的错误消息
func GetErrorMessage(code int16) string {
	switch code {
	case ErrInvalidData.Code:
		return ErrInvalidData.Message
	case ErrCourseNotFound.Code:
		return ErrCourseNotFound.Message
	case ErrInternalServer.Code:
		return ErrInternalServer.Message
	// 添加更多 case...
	default:
		return "Unknown error"
	}
}
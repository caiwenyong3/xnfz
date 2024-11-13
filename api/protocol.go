package protocol

const (
	Heartbeat         = int32(10001) + iota // 心跳请求
	HeartbeatResponse                       // 心跳响应
	ErrorMessage                            // 错误响应

	CourseDetail        = int32(20001) + iota // 课程明细进度响应
	CourseSelection                           // 课程选择请求
	CourseSelected                            // 课程选择响应
	CourseModeSelection                       // 课程模式选择请求
	CourseStart                               // 课程开始（课程模式选择响应）
	CourseEnd                                 // 课程结束
	CourseExit                                // 课程退出
	ObjectManipulation                        // 课程对象操作同步

)

type Message struct {
	Type       int32       `json:"type"`
	Code       int16       `json:"code"`
	DeviceCode string      `json:"deviceCode"`
	Data       interface{} `json:"data"`
}

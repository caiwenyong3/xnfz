package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string      `json:"type"`
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

func main() {
	// 构建 WebSocket URL
	u := url.URL{Scheme: "ws", Host: "47.98.211.153:9091", Path: "/ws", RawQuery: "main=true&deviceCode=123"}
	log.Printf("connecting to %s", u.String())

	// 建立 WebSocket 连接
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer c.Close()

	// 创建心跳定时器
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// 创建中断信号通道
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	// 启动心跳发送协程
	go func() {
		for {
			select {
			case <-ticker.C:
				heartbeat := Message{
					Type: "heartbeat",
					Data: "ping",
				}
				err := c.WriteJSON(heartbeat)
				if err != nil {
					log.Printf("heartbeat error: %v", err)
					close(done)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 启动消息接收协程
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				close(done)
				return
			}

			var response Message
			err = json.Unmarshal(message, &response)
			if err != nil {
				log.Println("json unmarshal error:", err)
				continue
			}

			switch response.Type {
			case "course_selected":
				log.Printf("recv: %s", message)
			case "course_start":
				log.Printf("recv: %s", message)
			case "heartbeat_response":
				// 心跳响应不打印日志
			case "object_manipulation":
				log.Printf("Received object_manipulation message: %s", message)
				// 这里可以添加处理 object_manipulation 消息的逻辑
				handleObjectManipulation(response.Data)
			case "course_exit":
				log.Printf("Received course_exit message: %s", message)
				// 添加处理课程退出的逻辑
			default:
				log.Printf("Received message of type: %s", response.Type)
			}
		}
	}()

	// 启动键盘输入处理协程
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		log.Println("Press '1' for course selection, '2' for course mode selection, '3' for object manipulation, '4' for course exit")

		for scanner.Scan() {
			input := scanner.Text()
			switch input {
			case "1":
				message := Message{
					Type: "course_selection",
					Data: map[string]string{"courseId": "1"},
				}
				err := c.WriteJSON(message)
				if err != nil {
					log.Printf("write error: %v", err)
					close(done)
					return
				}
				log.Println("course_selection request sent")

			case "2":
				message := Message{
					Type: "course_mode_selection",
					Data: map[string]interface{}{"courseId": "1", "mode": 3},
				}
				err := c.WriteJSON(message)
				if err != nil {
					log.Printf("write error: %v", err)
					close(done)
					return
				}
				log.Println("course_mode_selection request sent")

			case "3":
				// 示例 Unity 物体变换数据
				transformData := map[string]interface{}{
					"dataAttachment": map[int32]interface{}{
						1: map[string]interface{}{ // 物体 ID 为 1
							"position": map[string]float64{
								"x": 1.0,
								"y": 2.0,
								"z": 3.0,
							},
							"rotation": map[string]float64{
								"x": 0.0,
								"y": 90.0,
								"z": 0.0,
							},
							"scale": map[string]float64{
								"x": 1.0,
								"y": 1.0,
								"z": 1.0,
							},
						},
						2: map[string]interface{}{ // 物体 ID 为 2
							"position": map[string]float64{
								"x": 4.0,
								"y": 5.0,
								"z": 6.0,
							},
							"rotation": map[string]float64{
								"x": 0.0,
								"y": 180.0,
								"z": 0.0,
							},
							"scale": map[string]float64{
								"x": 2.0,
								"y": 2.0,
								"z": 2.0,
							},
						},
					},
				}

				message := Message{
					Type: "object_manipulation",
					Data: transformData,
				}
				err := c.WriteJSON(message)
				if err != nil {
					log.Printf("write error: %v", err)
					close(done)
					return
				}
				log.Println("object_manipulation request sent")

			case "4":
				message := Message{
					Type: "exit_course",
				}
				err := c.WriteJSON(message)
				if err != nil {
					log.Printf("write error: %v", err)
					close(done)
					return
				}
				log.Println("exit_course request sent")

			default:
				log.Println("Invalid input. Press '1' for course selection, '2' for course mode selection, '3' for object manipulation, '4' for course exit")
			}
		}
	}()

	// 主循环
	for {
		select {
		case <-done:
			log.Println("Connection closed")
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func handleObjectManipulation(data interface{}) {
	// 将 data 转换为 map[string]interface{}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		log.Println("Invalid data format for object_manipulation")
		return
	}

	// 获取 dataAttachment
	dataAttachment, ok := dataMap["dataAttachment"].(map[string]interface{})
	if !ok {
		log.Println("Invalid dataAttachment format")
		return
	}

	// 遍历所有物体
	for objectIDStr, objectData := range dataAttachment {
		objectID, err := strconv.ParseInt(objectIDStr, 10, 32)
		if err != nil {
			log.Printf("Invalid object ID: %s", objectIDStr)
			continue
		}

		objectMap, ok := objectData.(map[string]interface{})
		if !ok {
			log.Printf("Invalid object data format for object ID %d", objectID)
			continue
		}

		log.Printf("Object ID: %d", objectID)

		// 处理位置信息
		if position, ok := objectMap["position"].(map[string]interface{}); ok {
			log.Printf("  Position: x=%.2f, y=%.2f, z=%.2f",
				position["x"], position["y"], position["z"])
		}

		// 处理旋转信息
		if rotation, ok := objectMap["rotation"].(map[string]interface{}); ok {
			log.Printf("  Rotation: x=%.2f, y=%.2f, z=%.2f",
				rotation["x"], rotation["y"], rotation["z"])
		}

		// 处理缩放信息
		if scale, ok := objectMap["scale"].(map[string]interface{}); ok {
			log.Printf("  Scale: x=%.2f, y=%.2f, z=%.2f",
				scale["x"], scale["y"], scale["z"])
		}
	}
}

package main

import (
	"github.com/zxfonline/asyncmsg"
	"log"
	"time"
)

func main() {
	executor := asyncmsg.GExecutor()
	defer executor.Close()

	{
		log.Println("demo1")
		sessionId := asyncmsg.NewSession()
		data := []string{"demo1", "hello1", "hello2"}
		if taskService := asyncmsg.SendMsg(sessionId, data, func(params interface{}) interface{} { //params=SendMsg第二个参数
			log.Printf("demo1 req params=%+v", params)
			time.Sleep(2 * time.Second)
			return []string{"demo1", "world1", "world2"}
		}); taskService != nil {
			ret := asyncmsg.RecMsgWithTime(sessionId, 500*time.Millisecond).(*asyncmsg.PipeMsg) //阻塞等待消息返回
			if ret.Error != nil {
				log.Printf("demo1 ack:%+v", ret)
			} else {
				log.Printf("demo1 ack:%+v", ret)
			}
		}
	}
	{
		log.Println("demo2")
		sessionId := asyncmsg.NewSession()
		data := []string{"demo2", "hello1", "hello2"}
		if taskService := asyncmsg.SendMsg(sessionId, data, func(params interface{}) interface{} { //params=SendMsg第二个参数
			log.Printf("demo2 req params=%+v", params)
			return []string{"demo2", "world1", "world2"}
		}); taskService != nil {
			//default timeout 15 second
			ret := asyncmsg.RecMsg(sessionId).(*asyncmsg.PipeMsg) //阻塞等待消息返回
			if ret.Error != nil {
				log.Printf("demo2 %v", ret.Error)
			} else {
				log.Printf("demo2 ack:%+v", ret)
			}
		}
	}

	{
		log.Println("demo3")
		data := []string{"demo3", "hello1", "hello2"}
		asyncmsg.AsyncSendMsg(data, func(params interface{}) interface{} { //params=SendMsg第二个参数
			log.Printf("demo3 async params=%+v", params)
			return nil
		}, executor)
	}

}

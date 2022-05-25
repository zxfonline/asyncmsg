package asyncmsg

import (
	"fmt"
	"log"
	"time"
)

type Msg struct {
	SessionId int64
	Data      interface{}
	CallBack  CallBackMsg
}

//事件回调
type CallBackMsg func(interface{}) interface{}

func SendMsg(sessionID int64, data interface{}, callback CallBackMsg, executors ...Executor) *TaskService {
	task := NewTaskService(func(params ...interface{}) {
		msg := (params[0]).(Msg)
		if s := GetSession(msg.SessionId); s != nil { //没超时才会执行
			defer func() {
				if err := recover(); err != nil {
					s.Write(&PipeMsg{Error: fmt.Errorf("%v", err)})
				}
			}()
			s.Write(&PipeMsg{Data: msg.CallBack(msg.Data)})
		}
	}, Msg{SessionId: sessionID, Data: data, CallBack: callback})
	var executor Executor
	if len(executors) != 0 {
		executor = executors[0]
	} else {
		executor = GExecutor()
	}
	err := executor.Execute(task)
	if err != nil {
		log.Printf("excute task err:%v", err)
		return nil
	}
	return task
}

func AsyncSendMsg(data interface{}, callback CallBackMsg, executors ...Executor) *TaskService {
	task := NewTaskService(func(params ...interface{}) {
		msg := (params[0]).(Msg)
		defer func() {
			if err := recover(); err != nil {
				// log.Println(err)
				log.Printf("async send msg err:%v", err)
			}
		}()
		msg.CallBack(msg.Data)
	}, Msg{Data: data, CallBack: callback})
	var executor Executor
	if len(executors) != 0 {
		executor = executors[0]
	} else {
		executor = GExecutor()
	}
	err := executor.Execute(task)
	if err != nil {
		log.Printf("excute task err:%v", err)
		return nil
	}
	return task
}

func RecMsg(sId int64) interface{} {
	if s := GetSession(sId); s != nil {
		rt := s.Read(0)
		DelSession(sId)
		return rt
	}
	return nil
}

func RecMsgWithTime(sId int64, timeout time.Duration) interface{} {
	if s := GetSession(sId); s != nil {
		rt := s.Read(timeout)
		DelSession(sId)
		return rt
	}
	return nil
}

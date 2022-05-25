package asyncmsg

import (
	"errors"
	"sync"
	"time"
)

const (
	DEFAULT_ACCESS_TIMEOUT = 15 * time.Second
)

var (
	TimeoutErr = errors.New("pip wait time out")
)

type PipeMsg struct {
	Data  interface{}
	Error error
}
type Handler struct {
	Pipe chan *PipeMsg
}

type Session struct {
	Id      int64
	Handler *Handler
}

var (
	idPool   int64
	sessions map[int64]*Session
	m        *sync.RWMutex
)

func init() {
	sessions = make(map[int64]*Session)
	m = new(sync.RWMutex)
}

func (s *Session) Write(data *PipeMsg) {
	select {
	case s.Handler.Pipe <- data:
	default:
	}
}

func (s *Session) Read(timeout time.Duration) *PipeMsg {
	if timeout == 0 {
		timeout = DEFAULT_ACCESS_TIMEOUT
	}
	blockChan := time.NewTimer(timeout)
	select {
	case msg := <-s.Handler.Pipe:
		blockChan.Stop()
		return msg
	case <-blockChan.C:
		return &PipeMsg{Error: TimeoutErr} //Error为字符串错误码，外部可用于识别错误类型
	}
}

func NewSession() int64 {
	m.Lock()
	defer m.Unlock()
	h := Handler{
		Pipe: make(chan *PipeMsg, 1),
	}
	idPool++
	s := Session{
		Id:      idPool,
		Handler: &h,
	}
	sessions[s.Id] = &s
	return s.Id
}

func GetSession(id int64) *Session {
	m.RLock()
	defer m.RUnlock()
	if s, ok := sessions[id]; ok {
		return s
	}
	return nil
}

func DelSession(id int64) {
	m.Lock()
	defer m.Unlock()
	delete(sessions, id)
}

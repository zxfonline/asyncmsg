package asyncmsg

import (
	"container/list"
	"errors"
	"log"
	"time"

	"sync"
)

var (
	PoolStoppedError = errors.New("task pool stopped")
	PoolFullError    = errors.New("task pool full")
)

var (
	_GExecutor Executor
	onceInit   sync.Once
)

//GExecutor 全局任务执行器
func GExecutor() Executor {
	if _GExecutor == nil {
		onceInit.Do(func() {
			SetGExecutor(NewTaskPoolExcutor(8, 0x10000, false, 0))
		})
	}
	return _GExecutor
}

//SetGExecutor 设置全局任务执行器
func SetGExecutor(executor Executor) {
	if _GExecutor != nil {
		panic(errors.New("_GExecutor has been inited."))
	}
	_GExecutor = executor
}
func NewTaskExecutor(chanSize int) TaskExecutor {
	return make(chan *TaskService, chanSize)
}

type TaskExecutor chan *TaskService

func (c TaskExecutor) Close() {
	defer func() { recover() }()
	close(c)
}

//Execute 任务执行
func (c TaskExecutor) Execute(task *TaskService) (err error) {
	defer PanicToErr(&err)
	c <- task
	if wait := len(c); wait > cap(c)/10*5 && wait%100 == 0 {
		log.Printf("task executor taskChan process,waitChan:%d/%d", wait, cap(c))
	}
	return
}

//CallBack 事件回调
type CallBack func(...interface{})

//TaskService 执行器任务
type TaskService struct {
	callback CallBack
	args     []interface{}
	Cancel   bool //是否取消回调
}

//Call 代理执行
func (t *TaskService) Call() {
	if t.Cancel {
		return
	}
	defer func() {
		if e := recover(); e != nil {
			log.Printf("recover task service error:%v", e)
		}
	}()
	t.callback(t.args...)
}

//SetArgs 重置参数
func (t *TaskService) SetArgs(args ...interface{}) *TaskService {
	t.args = args
	return t
}

//SetArg 重置指定下标的参数
func (t *TaskService) SetArg(index int, arg interface{}) {
	if index < 0 || index+1 >= len(t.args) {
		return
	}
	t.args[index] = arg
}

//GetArg 获取指定下标的参数
func (t *TaskService) GetArg(index int) interface{} {
	if index < 0 || index+1 >= len(t.args) {
		return nil
	}
	return t.args[index]
}

//AddArgs 添加回调函数参数,startIndex<0表示顺序添加,startIndex>=0表示将参数从指定位置开始添加，原来位置的参数依次后移
func (t *TaskService) AddArgs(startIndex int, args ...interface{}) *TaskService {
	length := len(args)
	if length > 0 {
		stmp := t.args
		slenth := len(stmp)
		if startIndex < 0 {
			t.args = append(stmp, args...)
		} else if startIndex >= slenth {
			tl := startIndex + length
			temp := make([]interface{}, tl, tl)
			if slenth > 0 {
				copy(temp, stmp[0:slenth])
			}
			copy(temp[startIndex:], args)
			t.args = temp
		} else {
			tl := slenth + length
			temp := make([]interface{}, tl, tl)
			mv := stmp[startIndex:slenth]
			if startIndex > 0 {
				copy(temp, stmp[0:startIndex])
			}
			copy(temp[startIndex:startIndex+length], args)
			copy(temp[startIndex+length:], mv)
			t.args = temp
		}
	}
	return t
}

//NewTaskService 任务执行器任务
func NewTaskService(callback CallBack, params ...interface{}) *TaskService {
	length := len(params)
	temp := make([]interface{}, 0, length)
	temp = append(temp, params...)
	return &TaskService{callback: callback, args: temp}
}

//Executor 任务执行器
type Executor interface {
	//Execute 执行方法
	Execute(task *TaskService) error
	//Close 执行器关闭方法
	Close()
}

//任务执行器
type poolExecutor struct {
	closeD DoneChan
	wg     *sync.WaitGroup
}

func (c *poolExecutor) stop() {
	c.closeD.SetDone()
}

func (c *poolExecutor) execute(taskChan chan *TaskService, waitD DoneChan) {
	defer func() {
		c.wg.Done()
	}()
	for q := false; !q; {
		select {
		case <-c.closeD:
			q = true
		case task := <-taskChan:
			time.Sleep(2 * time.Second)
			task.Call()
		case <-waitD:
			select { //等待线程池剩余任务完成
			case task := <-taskChan:
				task.Call()
			default:
				q = true
			}
		}
	}
}

//MultiplePoolExecutor 并发任务执行器
type MultiplePoolExecutor struct {
	//任务缓冲池
	taskChan chan *TaskService
	//线程池执行关闭后是否立刻关闭子线程执行器
	shutdownNow bool
	//当 shutdownNow=false 时使用该变量 线程池执行关闭后等待子线程执行的时间，时间到后立刻关闭，当值为0时表示一直等待所有任务执行完毕
	shutdownWait time.Duration

	wgExecutor *sync.WaitGroup
	closeD     DoneChan
	waitD      DoneChan
	closeOnce  sync.Once
	//当前运行中的执行器 不用改队列
	executors *list.List
}

//NewTaskPoolExcutor 并发任务执行池
func NewTaskPoolExcutor(poolSize, chanSize uint, shutdownNow bool, shutdownWait time.Duration) Executor {
	wgExecutor := &sync.WaitGroup{}
	taskChan := make(chan *TaskService, chanSize)
	//	expvar.RegistChanMonitor("chanMultiExecutor", taskChan)
	waitD := NewDoneChan()
	p := &MultiplePoolExecutor{
		executors:    list.New(),
		taskChan:     taskChan,
		shutdownNow:  shutdownNow,
		shutdownWait: shutdownWait,
		waitD:        waitD,
		closeD:       NewDoneChan(),
		wgExecutor:   wgExecutor,
	}

	if poolSize < 1 {
		poolSize = 1
	}
	for i := 0; i < int(poolSize); i++ {
		ex := &poolExecutor{
			closeD: NewDoneChan(),
			wg:     wgExecutor,
		}
		p.executors.PushBack(ex)
		wgExecutor.Add(1)
		go ex.execute(taskChan, waitD)
	}
	return p
}

//Execute 执行方法
func (p *MultiplePoolExecutor) Execute(task *TaskService) error {
	select {
	case <-p.closeD:
		return PoolStoppedError
	default:
		p.taskChan <- task //阻塞等待
		if wait := len(p.taskChan); wait > cap(p.taskChan)/10*5 && wait%100 == 0 {
			log.Printf("taskPool executor taskChan process,waitChan:%d/%d", wait, cap(p.taskChan))
		}
		return nil
	}
}

func (p *MultiplePoolExecutor) waitDone() {
	//等待所有子任务执行执行完成
	p.wgExecutor.Wait()
	p.executors.Init()
}
func (p *MultiplePoolExecutor) clear() {
	for ex := p.executors.Front(); ex != nil; ex = ex.Next() {
		v := ex.Value.(*poolExecutor)
		v.stop()
	}
	p.waitDone()
}

//Close 执行器关闭方法
func (p *MultiplePoolExecutor) Close() {
	p.closeOnce.Do(func() {
		p.closeD.SetDone()
		if p.shutdownNow {
			p.clear()
		} else {
			if p.shutdownWait > 0 {
				time.AfterFunc(p.shutdownWait, func() {
					p.clear()
				})
			} else {
				p.waitD.SetDone()
				p.waitDone()
			}
		}
	})
}

package mocknet

import (
	"context"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
	"sync"
	"time"
)

type Ack struct {}

type Task struct {
	t time.Duration
	ch chan Ack
}

type NanoClock interface {
	// Time since start
	Now() time.Duration
}

type Environment interface {
	NanoClock
	ScheduleTask(delta time.Duration) (out chan Ack)
}

type ModelEnvironment struct {
	sync.RWMutex
	Queue *prque.Prque
	t time.Duration
	ctx context.Context
}

func NewModelEnvironment() *ModelEnvironment {
	return &ModelEnvironment{
		Queue: prque.New(),
	}
}

func (me *ModelEnvironment) ScheduleTask(delta time.Duration) (out chan Ack) {
	me.Lock()
	out = make(chan Ack)
	task := &Task{t: me.t + delta, ch: out}
	me.Queue.Push(task, -float32(task.t))
	me.Unlock()
	return
}

func (me *ModelEnvironment) Now() time.Duration {
	me.RLock()
	defer me.RUnlock()
	return me.t
}

func (me *ModelEnvironment) StepDelta(delta time.Duration) {
	me.Lock()
	defer me.Unlock()
	start := me.t
	end := start + delta
	for {
		if me.Queue.Empty() {
			break
		}
		nextItem := me.Queue.PopItem()
		nextTask := nextItem.(*Task)
		if nextTask.t >= end {
			me.Queue.Push(nextTask, -float32(nextTask.t))
			break
		}
		// acknowledge the task
		nextTask.ch <- struct{}{}
	}
	me.t = end
}

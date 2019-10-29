package mocknet

import (
	"context"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
	"sync"
	"time"
)

type TaskAck struct {}
type TaskSynAck chan TaskAck
type TaskSyn chan TaskSynAck

// The scheduler is build around Go channels, and replicates the SYN, SYN-ACK, ACK back and forth between the go routines:
//
// task scheduling: sending a SYN
// the scheduler having started a task: acknowledging a SYN with a SYN-ACK
// the task starts running: receiving the SYN-ACK
// the task completes: sends an ACK
// the scheduler knows a task was completed: an ACK is received

// Task has a time t it should execute at, and a channel to wait for a signal that the task is scheduled
// (from which the response can be used to communicate back that the task is completed)
type Task struct {
	t time.Duration
	ch TaskSyn
}

type NanoClock interface {
	// Time since start
	Now() time.Duration
}

type Environment interface {
	NanoClock
	SynTask(delta time.Duration) (out TaskSyn)
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

func (me *ModelEnvironment) SynTask(delta time.Duration) (out TaskSyn) {
	me.Lock()
	out = make(TaskSyn)
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
		sa := make(TaskSynAck)
		// syn-ack; acknowledge the syn (task scheduling)
		nextTask.ch <- sa
		// wait for ack: task complete
		<- sa
	}
	me.t = end
}

package workerext

import (
	"errors"
	"sync"
)

type Result interface{}

type Job interface {
	Go() Result
}

type Continue Job

type Work struct {
	args     []int
	workFunc func(args []int) Result
}

func (w *Work) Go() Result {
	return w.workFunc(w.args) // 由使用者定义 workFunc 返回的 Result 是单一结果还是链上下一项工作，具体判断在后面
}

type Stop struct {
	waiter *sync.WaitGroup
}

func (s *Stop) Go() Result {
	s.waiter.Done()
	return nil
}

type Worker struct {
	routines int
	jobs     chan Job
	ress     chan Result
}

func New() *Worker {
	return &Worker{}
}

func (w *Worker) Setup(numRoutine int) error {
	if w.jobs != nil {
		return errors.New("worker has had Job channel")
	}
	if w.ress != nil {
		return errors.New("worker has had Result channel")
	}

	return nil
}

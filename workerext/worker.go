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
	results  []Result
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

	w.routines = numRoutine
	w.jobs, w.ress = make(chan Job, 1), make(chan Result, 1)

	for i := 0; i < w.routines; i++ {
		go w.routine()
		// go w.result()
	}
	go w.result() // 只能有一个，否则会导致结束工作排到工作链后续工作前面

	return nil
}

func (w *Worker) routine() {
	for work := range w.jobs {
		w.ress <- work.Go()
	}
}

func (w *Worker) result() {
	for result := range w.ress { // 这里可能在通道关闭后还在运行
		//fmt.Printf("%T\n", result)
		cont, ok := result.(Continue) // 为了防止结束的工作排在工作链后续工作的前面，工作链必须连续串行
		for ok && cont != nil {
			//fmt.Println("result is a new Job")
			result = cont.Go()
			cont, ok = result.(Continue)
		}
		if result != nil {
			//fmt.Println("result is a new Int")
			w.results = append(w.results, result)
		}

		/*
			if cont, ok := result.(Continue); ok && cont != nil {
				w.ress <- cont.Go()
			} else {
				if result != nil {
					//fmt.Println("result is a new Int")
					w.results = append(w.results, result)
				}
			}
		*/
	}
}

func (w *Worker) Stop() error {
	if w.jobs == nil {
		return errors.New("worker has no Job channel")
	}
	if w.ress == nil {
		return errors.New("worker has no Result channel")
	}

	tick := &Stop{new(sync.WaitGroup)}

	tick.waiter.Add(w.routines)
	for i := 0; i < w.routines; i++ {
		w.jobs <- tick // 是不是应该放到 ress 里面
	}
	tick.waiter.Wait()

	close(w.jobs)
	close(w.ress)

	w.jobs, w.ress = nil, nil

	return nil
}

func (w *Worker) Do(work Job) {
	if w.jobs != nil {
		w.jobs <- work
	}
}

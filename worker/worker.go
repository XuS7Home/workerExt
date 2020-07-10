package worker

import (
	"errors"
	"sync"
)

// Result 为工作运行结果类型借口
type Result interface{}

// Job 为工作类型接口
// Go 方法为具体的工作内容，n 为工作的参数
type Job interface {
	// ** result Result 记录当前工作得到结果
	Go(n int) Result
}

// Continue 为 Job 别名，用于判断工作链得到的结果是否为新的工作
type Continue Job

// Worker 代表工人
type Worker struct {
	routines int         // 工人能够开始的工作例程数量
	jobs     chan Job    // 工人处于等待的工作通道
	ress     chan Result // 工人发送数据的结果通道
}

// stop 以 Job 的形式结束工人的工作
// waiter 为结束工作例程数量的计数器，全部结束说明工人停止工作
type stop struct {
	waiter *sync.WaitGroup
}

func (s *stop) Go(n int) Result { return nil } // ** 可以在里面设置 s.waiter.Done() 试试

// New 函数创建新的工人并返回工人的指针
func New() *Worker {
	return &Worker{}
}

// SpawnN 函数创建新的工人并进行初始化
// num 为工人能够开启的例程数量
func SpawnN(num int) *Worker {
	w := New()
	w.SpawnN(num)
	return w
}

// Worker.SpawnN 方法为工人更新例程数量、工作通道和结果通道，并开启所有工作例程
// num 是工人能够开启的例程总数
func (w *Worker) SpawnN(num int) error {
	if w.jobs != nil {
		return errors.New("worker have had Job channel")
	}
	if w.ress != nil {
		return errors.New("worker have had Result channel")
	}

	w.routines = num
	w.jobs, w.ress = make(chan Job, 1), make(chan Result, 1)

	for i := 0; i < w.routines; i++ {
		go w.routine(i) // ** 这里的 i 其实没有意义
	}

	return nil
}

// Worker.routine 方法是工人的一个被开启的例程，从工作通道中得到等待的工作并进行
func (w *Worker) routine(num int) { // ** num 没有意义
	for work := range w.jobs {
		if work != nil { // ** 有判断的必要吗
			var wg *sync.WaitGroup
			if stop, ok := work.(*stop); ok && stop != nil { // ** 这里的 && stop 可以去掉吧？用法要记住！
				wg = stop.waiter
			}

			w.ress <- work.Go(num)
			go w.advance(wg, num)

			if wg != nil {
				return
			}
		}
	}
}

// Worker.advance 将结果通道的工作链完成，并在 wg 存在时更新 wg 值
func (w *Worker) advance(wg *sync.WaitGroup, n int) { // n 没有意义
	if chainJob(<-w.ress, n); wg != nil {
		wg.Done()
	}
}

// chainJob 将一系列工作链完成
func chainJob(result Result, n int) Result {
	for result != nil {
		if j, ok := result.(Continue); ok && j != nil {
			result = j.Go(n)
		} else {
			break
		}
	}

	return result
}

// Worker.Stop 向工作通道中加入例程数量的 stop 工作，使各例程运行一个 stop 工作且停止
func (w *Worker) Stop() error {
	if w.jobs == nil {
		return errors.New("wroker has no job channel")
	}
	if w.ress == nil {
		return errors.New("worker has no result channel")
	}

	tick := &stop{new(sync.WaitGroup)}

	tick.waiter.Add(w.routines)
	for i := 0; i < w.routines; i++ {
		w.jobs <- tick
	}
	tick.waiter.Wait()

	close(w.jobs)
	close(w.ress) // 关闭通道能够使 range channel 跳出阻塞 ** 不过前面已经 return 了还有必要吗？可能是为了赋值 nil？

	w.jobs, w.ress = nil, nil

	return nil
}

// Worker.Do 向工作通道内加入工作
func (w *Worker) Do(work Job) {
	if w.jobs != nil {
		w.jobs <- work
	}
}

// Sentry 用于记录工人工作的结果
type Sentry struct {
	worker  *Worker
	mutex   *sync.Mutex
	waiter  *sync.WaitGroup
	results []Result
}

// guard 代表 Sentry 记录中的一项工作
type guard struct {
	sentry *Sentry
	job    Job
}

// unguard 代表 Sentry 记录中的一项结果
type unguard struct {
	sentry *Sentry
	result Result
}

// guard.Go 是工作的第一步运行
func (g *guard) Go(n int) Result {
	return &unguard{g.sentry, g.job.Go(n)}
}

// unguard.Go 是工作的完全运行得到结果
func (u *unguard) Go(n int) Result {
	result := chainJob(u.result, n)
	u.sentry.mutex.Lock()
	u.sentry.results = append(u.sentry.results, result)
	u.sentry.mutex.Unlock()
	u.sentry.waiter.Done()
	return result
}

// Worker.Sentry 对已有工人设置监听器用于记录
func (w *Worker) Sentry() *Sentry {
	if w.jobs == nil || w.ress == nil {
		return nil
	}

	sentry := &Sentry{
		worker: w,
		mutex:  new(sync.Mutex),
		waiter: new(sync.WaitGroup),
	}

	return sentry
}

// Sentry.Guard 开启对一项工作的记录
func (s *Sentry) Guard(j Job) {
	s.waiter.Add(1) // ** 为什么本身 worker 没有 waitGroup 呢？气抖冷！
	s.worker.Do(&guard{s, j})
}

// Sentry.Wait 等待所有被监听工作结束
func (s *Sentry) Wait() (result []Result) {
	s.waiter.Wait()
	return s.results
}

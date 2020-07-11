package workerext

import "testing"

func TestWorker(t *testing.T) {
	addNumber := new(Work)
	checkResult := new(Work)

	addNumber.args = []int{1, 2, 3}
	addNumber.workFunc = func(args []int) Result {
		answer := 0
		for _, val := range args {
			answer += val
		}

		checkResult.args = []int{answer}
		return checkResult
	}

	checkResult.workFunc = func(args []int) Result {
		return int(args[0] / 3)
	}

	work := addNumber
	worker, times := New(), 1000
	worker.Setup(3)

	for i := 0; i < times; i++ {
		worker.Do(work)
	}

	worker.Stop()

	if len(worker.results) != times {
		t.Errorf("wrong results length: %v", len(worker.results))
	}

	for i := 0; i < times; i++ {
		if worker.results[i].(int) != 2 {
			t.Errorf("wrong results value: %v", worker.results[i])
		}
	}
}

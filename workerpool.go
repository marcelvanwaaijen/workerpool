package workerpool

import (
	"fmt"
	"sync"
)

type WorkerPool[T any] struct {
	wg    sync.WaitGroup
	wp    chan interface{}
	queue chan T
	fn    func(T) error
	abort bool
	err   error
	m     sync.RWMutex
}

// NewPool initializes a new WorkerPool with the specified number of max concurrent workers and the
// max queue length of the waiting queue and finally the function definition to execute for each payload.
// When the function definition returns an error, the workerpool will be stopped and the Wait function will
// return the error.
func NewPool[T any](workers, queuelength int, task func(param T) error) *WorkerPool[T] {
	return &WorkerPool[T]{wp: make(chan interface{}, workers), queue: make(chan T, queuelength), fn: task, abort: false}
}

// SetTask can be used to change the function used for executing the payload
func (wp *WorkerPool[T]) SetTask(fn func(param T) error) {
	wp.m.Lock()
	defer wp.m.Unlock()
	wp.fn = fn
}

// Add payload to the worker pool
func (wp *WorkerPool[T]) Add(param T) error {
	wp.m.RLock()
	defer wp.m.RUnlock()

	if wp.abort {
		return fmt.Errorf("work aborted, can't add more work")
	}
	if wp.err != nil {
		return fmt.Errorf("error found, can't add more work")
	}
	wp.queue <- param
	return nil
}

// Start executing tasks as soon as payload has been added
func (wp *WorkerPool[T]) Start() {
	go func() {
		for it := range wp.queue {
			wp.m.RLock()
			// if the workerpool is aborted, stop the execution
			if wp.abort {
				wp.m.RUnlock()
				return
			}

			// if the last call to the task resulted in an error, stop the execution
			if wp.err != nil {
				wp.m.RUnlock()
				return
			}
			wp.m.RUnlock()

			wp.wg.Add(1)
			wp.wp <- true
			go func(it T, wg *sync.WaitGroup) {
				defer wg.Done()
				defer func() { <-wp.wp }()

				// if the execution of the specified Task returns an error, stop the execution and let the Wait function return the error
				if err := wp.fn(it); err != nil {
					wp.m.Lock()
					defer wp.m.Unlock()

					if wp.err == nil {
						close(wp.queue)
						wp.err = err
					}
				}

			}(it, &(wp.wg))
		}
	}()
}

// Elegantly abort the queued work on the worker pool.
// Any running tasks will finish before stopping but no additional
// payload can be added as soon as Abort is called
func (wp *WorkerPool[T]) Abort() {
	wp.m.Lock()
	defer wp.m.Unlock()

	close(wp.queue)
	wp.abort = true
}

// Wait for the workerpool to finish all pending tasks
func (wp *WorkerPool[T]) Wait() error {
	wp.wg.Wait()

	// if the last call to the task function resulted in an error, return that error
	return wp.err
}

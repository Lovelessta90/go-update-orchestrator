package pool

import (
	"context"
	"sync"
)

// WorkerPool manages a bounded pool of workers for concurrent execution.
type WorkerPool struct {
	maxWorkers int
	taskQueue  chan Task
	wg         sync.WaitGroup
}

// Task represents a unit of work.
type Task func(ctx context.Context) error

// New creates a new worker pool with the specified number of workers.
func New(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan Task, maxWorkers),
	}
}

// Start initializes the worker pool and starts all workers.
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

// Submit adds a task to the pool for execution.
func (p *WorkerPool) Submit(task Task) {
	p.taskQueue <- task
}

// Stop gracefully shuts down the worker pool.
func (p *WorkerPool) Stop() {
	close(p.taskQueue)
	p.wg.Wait()
}

// worker is the main worker goroutine that processes tasks.
func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()

	for task := range p.taskQueue {
		if err := task(ctx); err != nil {
			// TODO: Handle error (logging, metrics, etc)
		}
	}
}

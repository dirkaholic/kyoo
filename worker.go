package jobqueue

import (
	"sync"
)

// Worker - the worker threads that actually process the jobs
type Worker struct {
	done             *sync.WaitGroup
	readyPool        chan chan Job
	assignedJobQueue chan Job

	quit chan bool
}

// NewWorker creates a new worker
func NewWorker(readyPool chan chan Job, done *sync.WaitGroup) *Worker {
	return &Worker{
		done:             done,
		readyPool:        readyPool,
		assignedJobQueue: make(chan Job),
		quit:             make(chan bool),
	}
}

// Start - begins the job processing loop for the worker
func (w *Worker) Start() {
	go func() {
		w.done.Add(1)
		for {
			w.readyPool <- w.assignedJobQueue // check the job queue in
			select {
			case job := <-w.assignedJobQueue: // see if anything has been assigned to the queue
				job.Process()
			case <-w.quit:
				w.done.Done()
				return
			}
		}
	}()
}

// Stop stops the worker
func (w *Worker) Stop() {
	w.quit <- true
}

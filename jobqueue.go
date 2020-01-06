package jobqueue

import (
	"sync"
)

// JobQueue represents a queue for enqueueing jobs to be processed
type JobQueue struct {
	internalQueue     chan Job
	readyPool         chan chan Job
	workers           []*Worker
	dispatcherStopped *sync.WaitGroup
	workersStopped    *sync.WaitGroup
	quit              chan bool
	stopped           bool
	stoppedMutex      *sync.Mutex
}

// NewJobQueue creates a new job queue
func NewJobQueue(maxWorkers int) *JobQueue {
	workersStopped := &sync.WaitGroup{}
	readyPool := make(chan chan Job, maxWorkers)
	workers := make([]*Worker, maxWorkers, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workers[i] = NewWorker(readyPool, workersStopped)
	}
	return &JobQueue{
		internalQueue:     make(chan Job),
		readyPool:         readyPool,
		workers:           workers,
		dispatcherStopped: &sync.WaitGroup{},
		workersStopped:    workersStopped,
		quit:              make(chan bool),
	}
}

// Start starts the workers and dispatcher
func (q *JobQueue) Start() {
	for i := 0; i < len(q.workers); i++ {
		q.workers[i].Start()
	}
	go q.dispatch()

	q.stoppedMutex = &sync.Mutex{}
	q.setStopped(false)
}

// Stop stops the workers and dispatcher
// Also closes internal job queue channel after the last value has been received
func (q *JobQueue) Stop() {
	q.setStopped(true)

	// Stopping queue
	q.quit <- true
	q.dispatcherStopped.Wait()

	// Stopped queue
	close(q.internalQueue)
}

// Stopped indicates wether the queue is stopped
func (q *JobQueue) Stopped() bool {
	return q.getStopped()
}

func (q *JobQueue) setStopped(s bool) {
	q.stoppedMutex.Lock()
	q.stopped = s
	q.stoppedMutex.Unlock()
}

func (q *JobQueue) getStopped() bool {
	q.stoppedMutex.Lock()
	s := q.stopped
	q.stoppedMutex.Unlock()

	return s
}

func (q *JobQueue) dispatch() {
	q.dispatcherStopped.Add(1)
	for {
		select {
		case job := <-q.internalQueue: // we got something in on our queue
			workerChannel := <-q.readyPool // check out an available worker
			workerChannel <- job           // send the request to the channel
		case <-q.quit:
			for i := 0; i < len(q.workers); i++ {
				q.workers[i].Stop()
			}
			q.workersStopped.Wait()
			q.dispatcherStopped.Done()
			return
		}
	}
}

// Submit submits a new job to be processed to the internal queue
func (q *JobQueue) Submit(job Job) {
	if !q.getStopped() {
		q.internalQueue <- job
	}
}

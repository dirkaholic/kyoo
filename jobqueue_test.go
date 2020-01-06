package jobqueue_test

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	jobqueue "github.com/dirkaholic/kyoo"
	"github.com/dirkaholic/kyoo/job"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type JobQueueSuite struct{}

var _ = Suite(&JobQueueSuite{})

func (s *JobQueueSuite) TestJobQueueStartsAndStopsSuccesfully(c *C) {
	queue := jobqueue.NewJobQueue(runtime.NumCPU())
	queue.Start()

	c.Assert(queue.Stopped(), Equals, false)

	queue.Stop()
	c.Assert(queue.Stopped(), Equals, true)
}

func (s *JobQueueSuite) TestJobQueueExecutesFunctionSuccessfully(c *C) {
	queue := jobqueue.NewJobQueue(runtime.NumCPU())
	queue.Start()

	funcExecuted := false

	queue.Submit(&job.FuncExecutorJob{Func: func() error {
		funcExecuted = true
		return nil
	}})

	queue.Stop()
	c.Assert(funcExecuted, Equals, true)
}

func (s *JobQueueSuite) TestJobQueueSetsJobErrorWhenErrorOccursDuringExecution(c *C) {
	queue := jobqueue.NewJobQueue(runtime.NumCPU())
	queue.Start()

	job := &job.FuncExecutorJob{Func: func() error {
		return errors.New("Error while executing function")
	}}

	c.Assert(job.Err, IsNil)
	queue.Submit(job)

	queue.Stop()
	c.Assert(job.Err, NotNil)
}

func (s *JobQueueSuite) TestAddingManyJobsWithLowNumberOfWorkersAvailable(c *C) {
	queue := jobqueue.NewJobQueue(2)
	queue.Start()

	jobCount := 10000
	processCount := 0
	processCountMutex := &sync.Mutex{}

	for index := 0; index < jobCount; index++ {
		queue.Submit(&job.FuncExecutorJob{Func: func() error {
			processCountMutex.Lock()
			processCount++
			processCountMutex.Unlock()
			return nil
		}})
	}

	queue.Stop()
	c.Assert(processCount, Equals, jobCount)
}

func (s *JobQueueSuite) TestPendingJobsAreExecutedWhenQueueIsStopped(c *C) {
	queue := jobqueue.NewJobQueue(runtime.NumCPU())
	queue.Start()

	jobCount := 100
	processCount := 0
	processCountMutex := &sync.Mutex{}

	for index := 0; index < jobCount; index++ {
		queue.Submit(&job.FuncExecutorJob{Func: func() error {
			time.Sleep(10 * time.Millisecond) // add a little delay so the queue is stopped before all jobs are finished
			processCountMutex.Lock()
			processCount++
			processCountMutex.Unlock()
			return nil
		}})
	}

	queue.Stop()
	c.Assert(processCount, Equals, jobCount)
}

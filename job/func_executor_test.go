package job_test

import (
	"errors"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/dirkaholic/kyoo/job"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type JobSuite struct{}

var _ = Suite(&JobSuite{})

func (s *JobSuite) TestProcessingJobExecutesFunctionSuccessfully(c *C) {
	funcExecuted := false

	job := &job.FuncExecutorJob{Func: func() error {
		funcExecuted = true
		return nil
	}}

	job.Process()
	c.Assert(job.Err, IsNil)
	c.Assert(funcExecuted, Equals, true)
}

func (s *JobSuite) TestErrorDuringJobProcessingSetsJobError(c *C) {
	job := &job.FuncExecutorJob{Func: func() error {
		return errors.New("Error while executing function")
	}}

	job.Process()
	c.Assert(job.Err, NotNil)
}

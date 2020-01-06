package job

// FuncExecutorJob represents a job that executes a generic function defined by the caller
type FuncExecutorJob struct {
	Err  error
	Func func() error
}

// Process processes the function defined for the job
func (job *FuncExecutorJob) Process() {
	job.Err = job.Func()
}

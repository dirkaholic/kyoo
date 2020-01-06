package jobqueue

// Job interface for job processing
type Job interface {
	Process()
}

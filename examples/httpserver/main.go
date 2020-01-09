package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	jobqueue "github.com/dirkaholic/kyoo"
	"github.com/dirkaholic/kyoo/job"
)

var queue *jobqueue.JobQueue

func handler(w http.ResponseWriter, r *http.Request) {
	queue.Submit(&job.FuncExecutorJob{Func: func() error {
		return doTheHeavyBackgroundWork(r.URL.Path)
	}})
	fmt.Printf("%s - submitted %s !!\n", time.Now().String(), r.URL.Path)

	fmt.Fprint(w, "Job added to queue.")
}

func main() {
	queue = jobqueue.NewJobQueue(runtime.NumCPU() * 2)
	queue.Start()

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func doTheHeavyBackgroundWork(path string) error {
	time.Sleep(2 * time.Second)
	fmt.Printf("%s - processed %s !!\n", time.Now().String(), path)
	return nil
}

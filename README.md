# kyoo: A Go library providing an unlimited job queue and concurrent worker pools

![Build](https://gitlab.com/dirkaholic/kyoo/badges/master/pipeline.svg) ![Coverage](https://gitlab.com/dirkaholic/kyoo/badges/master/coverage.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/dirkaholic/kyoo)](https://goreportcard.com/report/github.com/dirkaholic/kyoo)

## About

kyoo is the phonetic transcription of the word queue. It provides a job queue that can hold as much jobs as
resources are available on the running system.

The queue has the following characteristics:
* No limit of jobs to be queued (only limited by system resources = memory)
* Concurrent processing of jobs using worker pools
* When stopping queue, pending jobs are still processed

The library contains a simple `Job` interface and a simple `FuncExecutorJob` that
just executes a given function and implements that interface. With that nearly all
kinds of workloads should be processable already but of course it is possible to add
custom implementations of the `Job` interface.

Possible use cases for the library are:
* Consumers for message queues like RabbitMQ or Amazon SQS
* Processing web server requests offloading time extensive work into background jobs
* All kinds of backend processing jobs like image optimization, etc.

## Example

The following example shows a simple http server offloading jobs to the jobqueue that is constantly
processed in the background.

```go
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
```

Test the offloading by sending a bunch of http requests to the server 

```bash
$ for i in {1..10}; do http http://127.0.0.1:8080/test/$i; done
````

The output on http server side should be similar like this

```bash
2020-01-09 21:36:36.156277 +0100 CET m=+5.733617272 - submitted /test/1 !!
2020-01-09 21:36:36.443521 +0100 CET m=+6.020861136 - submitted /test/2 !!
2020-01-09 21:36:36.730535 +0100 CET m=+6.307874793 - submitted /test/3 !!
2020-01-09 21:36:37.021405 +0100 CET m=+6.598744533 - submitted /test/4 !!
2020-01-09 21:36:37.311973 +0100 CET m=+6.889312431 - submitted /test/5 !!
2020-01-09 21:36:37.609868 +0100 CET m=+7.187208115 - submitted /test/6 !!
2020-01-09 21:36:37.895222 +0100 CET m=+7.472561850 - submitted /test/7 !!
2020-01-09 21:36:38.160524 +0100 CET m=+7.737863891 - processed /test/1 !!
2020-01-09 21:36:38.171491 +0100 CET m=+7.748830724 - submitted /test/8 !!
2020-01-09 21:36:38.445832 +0100 CET m=+8.023171514 - processed /test/2 !!
2020-01-09 21:36:38.448423 +0100 CET m=+8.025762679 - submitted /test/9 !!
2020-01-09 21:36:38.730541 +0100 CET m=+8.307880933 - submitted /test/10 !!
2020-01-09 21:36:38.735158 +0100 CET m=+8.312497505 - processed /test/3 !!
2020-01-09 21:36:39.024788 +0100 CET m=+8.602128093 - processed /test/4 !!
2020-01-09 21:36:39.315991 +0100 CET m=+8.893331115 - processed /test/5 !!
2020-01-09 21:36:39.614848 +0100 CET m=+9.192187633 - processed /test/6 !!
2020-01-09 21:36:39.896692 +0100 CET m=+9.474031970 - processed /test/7 !!
2020-01-09 21:36:40.175952 +0100 CET m=+9.753291345 - processed /test/8 !!
2020-01-09 21:36:40.451877 +0100 CET m=+10.029216847 - processed /test/9 !!
2020-01-09 21:36:40.734289 +0100 CET m=+10.311628415 - processed /test/10 !!
```

## More examples

* [SQS worker](examples/sqsworker)

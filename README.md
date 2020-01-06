# kyoo: A Go library providing an unlimited job queue and concurrent worker pools

![Build](https://gitlab.com/dirkaholic/kyoo/badges/master/pipeline.svg) ![Coverage](https://gitlab.com/dirkaholic/kyoo/badges/master/coverage.svg)

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
* Processing web server requests
* All kinds of backend processing jobs like image optimization, etc.

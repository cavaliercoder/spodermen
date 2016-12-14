package main

// TODO: implement keep-alive

import (
	"fmt"
	"os"
	"time"
)

const (
	WORKERS    = 4
	POLL_SLEEP = 3 * time.Second
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(usage(1))
	}

	// history of visited uris
	history := make(map[string]int, 4096)

	// queue of all found hrefs
	triage := make(chan *CrawlRequest, 4096)
	defer close(triage)

	// queue of unique hrefs
	ready := make(chan *CrawlRequest, 4096)
	defer close(ready)

	// semaphore of requests in flight (needed because requests are dequeued
	// when they start; not when they finish)
	queued := make(chan int, 4096)
	defer close(queued)

	// start a loop to filter for unique hrefs
	go func() {
		for req := range triage {
			uri := req.URL.String()
			if _, ok := history[uri]; ok {
				history[uri]++
			} else {
				history[uri] = 1
				queued <- 1
				go func(req *CrawlRequest) {
					ready <- req
				}(req)
			}
		}
	}()

	// async func to enqueue a found href for triage
	enqueue := func(a ...*CrawlRequest) {
		go func() {
			for _, req := range a {
				triage <- req
			}
		}()
	}

	// start workers
	for i := 0; i < WORKERS; i++ {
		go func(i int) {
			printf("Starting worker %d\n", i+1)
			for {
				req := <-ready
				ch := crawl(req)
				for resp := range ch {
					enqueue(resp)
				}
				queued <- -1
			}
		}(i)
	}

	// enqueue entry point
	req, err := NewCrawlRequest(os.Args[1])
	panicOn(err)
	enqueue(req)

	inFlight := 0
	done := make(chan bool)
	go func() {
		for {
			inFlight += <-queued
			if inFlight == 0 {
				break
			}
		}
		close(done)
	}()

	<-done

	printf("Dolan, y u do dis?\n")
}

func usage(exitCode int) int {
	w := os.Stdout
	if exitCode > 0 {
		w = os.Stderr
	}

	fmt.Fprintf(w, "usage: %s [url]\n", os.Args[0])
	return exitCode
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func printf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, format, a...)
}

func dprintf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "DEBUG: %v", fmt.Sprintf(format, a...))
}

func errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

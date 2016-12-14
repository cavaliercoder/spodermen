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

	panicOn(initOutput())

	crawler := NewCrawler()

	// start workers
	for i := 0; i < WORKERS; i++ {
		go func(i int) {
			printf("Starting worker %d\n", i+1)
			for {
				resp, err := crawler.Next()
				if err != nil {
					errorf("%v\n", err)
				} else {
					printf("%v\n", resp)
				}
			}
		}(i)
	}

	// enqueue entry point
	req, err := NewCrawlRequest(os.Args[1])
	panicOn(err)

	crawler.Start(req)

	for {
		time.Sleep(3 * time.Second)
	}

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

package main

// TODO: implement keep-alive

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/urfave/cli.v1"
)

const (
	PackageName    = "spodermen"
	PackageVersion = "1.0.0"
)

const (
	POLL_SLEEP = 3 * time.Second
)

func main() {
	app := cli.NewApp()
	app.Name = PackageName
	app.Version = PackageVersion
	app.Usage = "A dumb site crawler to highlight broken links and faulty routes"
	app.Action = crawl
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file,f",
			Usage: "read URLs from `FILE`",
		},
		cli.StringFlag{
			Name:  "prefix,p",
			Usage: "prefix all URLs with `PREF`",
		},
		cli.IntFlag{
			Name:  "workers,w",
			Usage: "worker count",
			Value: runtime.NumCPU(),
		},
		cli.BoolFlag{
			Name:  "no-follow",
			Usage: "don't follow links",
		},
	}

	app.Run(os.Args)
}

func crawl(c *cli.Context) error {
	// urls is the channel that receives all URLs to be crawled
	urls := make(chan string, 4096)
	if path := c.String("file"); path != "" {
		// read URL list from a text file
		v, err := loadURLFile(path, c.String("prefix"))
		if err != nil {
			return err
		}
		for _, u := range v {
			urls <- u
		}
	} else {
		// read URL list from Args
		for _, u := range c.Args() {
			urls <- u
		}
	}

	opts := &CrawlOptions{
		NoFollow: c.Bool("no-follow"),
	}

	var waiting int32
	workers := c.Int("workers")
	reqQueue := make(chan *CrawlRequest, 4096)

	// start producer
	go func() {
		stats := make(map[string]int)
		for {
			// break if queue is empty and all workers are waiting
			// TODO: prevent early exit when using only one worker.
			v := atomic.LoadInt32(&waiting)
			l := len(urls)
			if int(v) == workers && l == 0 {
				close(reqQueue)
				break
			}

			select {
			case url := <-urls:
				// ignore URL fragment
				if i := strings.Index(url, "#"); i != -1 {
					url = url[:i]
				}

				// dedupe requests
				stat := stats[url]
				if stat > 0 {
					continue
				}
				stats[url]++

				req, err := NewCrawlRequest(url, nil)
				if err != nil {
					panic(err) // TODO
				}
				reqQueue <- req
			default:
			}
		}
	}()

	// start workers
	wg := &sync.WaitGroup{}
	wg.Add(workers)
	crawler := NewCrawler(opts)
	for i := 0; i < workers; i++ {
		time.Sleep(time.Millisecond * 100) // slow start
		go func(i int) {
			//printf("Starting worker %d\n", i+1)
			for {
				atomic.AddInt32(&waiting, 1)
				req := <-reqQueue
				atomic.AddInt32(&waiting, -1)
				if req == nil {
					break
				}

				resp, err := crawler.Do(req)
				if err != nil {
					errorf("%v\n", err)
				} else {
					if len(resp.URLs) > 0 {
						for _, u := range resp.URLs {
							urls <- u
						}
					}
					printf("%-26s %v\n", time.Now().Format("2006-01-02 15:04:05.999999"), resp)
				}
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	printf("Dolan, y u do dis?\n")

	return nil
}

// loadURLFile reads a list of URLs from a text file; one URL per line.
func loadURLFile(path, prefix string) ([]string, error) {
	var err error
	f := os.Stdin
	if path != "-" {
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}

	urls := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		urls = append(urls, prefix+scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
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

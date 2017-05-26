package main

// TODO: implement keep-alive

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"net/url"

	"gopkg.in/urfave/cli.v1"
)

const (
	PackageName    = "spodermen"
	PackageVersion = "1.0.0"
)

var (
	stopFlag int32
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
		cli.StringSliceFlag{
			Name:  "follow-hosts,a",
			Usage: "allow links on additional hosts",
		},
		cli.BoolFlag{
			Name:  "no-follow",
			Usage: "don't follow links",
		},
	}

	handleSignals()
	app.Run(os.Args)
}

func crawl(c *cli.Context) error {
	opts := &CrawlOptions{
		NoFollow: c.Bool("no-follow"),
		Hosts:    make(map[string]bool),
	}

	for _, s := range c.StringSlice("follow-hosts") {
		opts.Hosts[s] = true
	}

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
			x, err := url.Parse(u)
			if err != nil {
				return err
			}

			opts.Hosts[x.Host] = true
			urls <- u
		}
	}

	var waiting int32
	workers := c.Int("workers")
	reqQueue := make(chan *CrawlRequest, 4096)

	// start producer
	go func() {
		seen := make(map[string]int)
		for {
			// break if signalled
			if v := atomic.LoadInt32(&stopFlag); v != 0 {
				close(reqQueue)
				break
			}

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
				count := seen[url]
				if count > 0 {
					continue
				}
				seen[url]++

				req, err := NewCrawlRequest(url, !opts.NoFollow)
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
	crawler := NewCrawler(opts)
	for i := 0; i < workers; i++ {
		time.Sleep(time.Millisecond * 100) // slow start
		if v := atomic.LoadInt32(&stopFlag); v != 0 {
			break
		}

		go func(i int) {
			wg.Add(1)
			defer wg.Done()
			for {
				// break if signalled
				// TODO: use Context to stop requests in flight
				if v := atomic.LoadInt32(&stopFlag); v != 0 {
					break
				}

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
		}(i)
	}

	wg.Wait()
	printf("Dolan, y u do dis?\n")
	printf("%v\n", crawler.Stats().JSON())

	return nil
}

func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, os.Kill)

	go func() {
		for s := range ch {
			v := atomic.AddInt32(&stopFlag, 1)
			if v > 1 {
				fmt.Fprintf(os.Stderr, "Caught %v - forcing exit...\n", s)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Caught %v - cleaning up...\n", s)
			// TODO: sleep and force kill
		}
	}()
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

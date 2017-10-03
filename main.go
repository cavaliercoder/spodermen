package main

// TODO: implement keep-alive

import (
	"bufio"
	"context"
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

var (
	ctx    context.Context
	cancel context.CancelFunc
)

func main() {
	ctx, cancel = context.WithCancel(context.Background())

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
	q := NewCrawlRequestQueue(4096, ctx)

	opts := &CrawlOptions{
		NoFollow: c.Bool("no-follow"),
		Hosts:    make(map[string]bool),
	}

	for _, s := range c.StringSlice("follow-hosts") {
		opts.Hosts[s] = true
	}

	if path := c.String("file"); path != "" {
		// read URL list from a text file
		v, err := loadURLFile(path, c.String("prefix"))
		if err != nil {
			return err
		}
		for _, u := range v {
			req, err := NewCrawlRequest(u, !opts.NoFollow)
			if err != nil {
				panic(err) // TODO
			}
			q.Enqueue(req)
		}
	} else {
		// read URL list from Args
		for _, u := range c.Args() {
			x, err := url.Parse(u)
			if err != nil {
				return err
			}

			opts.Hosts[x.Host] = true

			req, err := NewCrawlRequest(u, !opts.NoFollow)
			if err != nil {
				panic(err) // TODO
			}
			q.Enqueue(req)
		}
	}

	workers := c.Int("workers")

	// start workers
	wg := &sync.WaitGroup{}
	crawler := NewCrawler(opts, ctx)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				req, ok := q.Dequeue()
				if !ok {
					break
				}

				resp, err := crawler.Do(req)
				if err != nil {
					errorf("%v\n", err)
					q.Done()
					continue
				}

				reqs := make([]*CrawlRequest, 0, len(resp.URLs))
				for _, u := range resp.URLs {
					if i := strings.Index(u, "#"); i != -1 {
						u = u[:i] // ignore URL fragment
					}
					req, err := NewCrawlRequest(u, !opts.NoFollow)
					if err != nil {
						panic(err) // TODO
					}
					reqs = append(reqs, req)
				}
				q.Enqueue(reqs...)
				printf("%-26s %v\n", time.Now().Format("2006-01-02 15:04:05.999999"), resp)
				q.Done()
			}
		}(i)
		//time.Sleep(time.Millisecond * 100) // slow start
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
			cancel()
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

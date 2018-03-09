package main

// TODO: implement keep-alive

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/cavaliercoder/spodermen/crawler"
	"gopkg.in/urfave/cli.v1"
)

const (
	PackageName    = "spodermen"
	PackageVersion = "1.0.0"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
)

// UserAgent is the UserAgent header set for all HTTP requests.
var UserAgent = fmt.Sprintf("%s bot/%s", PackageName, PackageVersion)

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
	// configure crawler
	opts := []crawler.Option{
		crawler.UserAgentString(UserAgent),
		crawler.WithContext(ctx),
	}
	noFollow := c.Bool("no-follow")
	if noFollow {
		opts = append(opts, crawler.NoFollow())
	}
	if hosts := c.StringSlice("follow-hosts"); hosts != nil {
		opts = append(opts, crawler.FollowHosts(hosts...))
	}

	// seed request queue
	q := crawler.NewQueue(4096, ctx)
	if path := c.String("file"); path != "" {
		// read URL list from a text file
		urls, err := loadURLFile(path, c.String("prefix"))
		if err != nil {
			return err
		}
		go func(urls []string) {
			for _, u := range urls {
				req, err := crawler.NewRequest(u, !noFollow)
				if err != nil {
					panic(err) // TODO
				}
				q.Enqueue(req)
			}
		}(urls)
		time.Sleep(time.Second / 5) // seed queue
	} else {
		// read URL list from Args
		for _, u := range c.Args() {
			x, err := url.Parse(u)
			if err != nil {
				return err
			}
			opts = append(opts, crawler.FollowHosts(x.Host))
			req, err := crawler.NewRequest(u, !noFollow)
			if err != nil {
				panic(err) // TODO
			}
			q.Enqueue(req)
		}
	}

	// start workers
	workers := c.Int("workers")
	crawler, err := crawler.New(opts...)
	if err != nil {
		panic(err)
	}
	stats := crawler.Crawl(q, workers)
	fmt.Println("Dolan, y u do dis?")
	fmt.Println(stats.JSON())

	return nil
}

func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, os.Kill)
	go func() {
		v := 0
		for s := range ch {
			v++
			if v > 1 {
				fmt.Fprintf(os.Stderr, "Caught %v - forcing exit...\n", s)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Caught %v - cleaning up...\n", s)
			cancel()
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

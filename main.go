package main

// TODO: implement keep-alive

import (
	"bufio"
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"os"
	"time"
)

const (
	PACKAGE_NAME    = "spodermen"
	PACKAGE_VERSION = "1.0.0"
)

const (
	WORKERS    = 4
	POLL_SLEEP = 3 * time.Second
)

func main() {
	app := cli.NewApp()
	app.Name = PACKAGE_NAME
	app.Version = PACKAGE_VERSION
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
		cli.BoolFlag{
			Name:  "no-follow",
			Usage: "don't follow links",
		},
	}

	app.Run(os.Args)
}

func crawl(c *cli.Context) error {
	// init output templates
	if err := initOutput(); err != nil {
		return err
	}

	urls := []string{}
	if path := c.String("file"); path != "" {
		if v, err := loadURLFile(path, c.String("prefix")); err != nil {
			return err
		} else {
			urls = v
		}
	} else {
		urls = []string(c.Args())
	}

	opts := &CrawlOptions{
		NoFollow: c.Bool("no-follow"),
	}

	// start workers
	crawler := NewCrawler(opts)
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

	reqs := make([]*CrawlRequest, len(urls))
	for i, u := range urls {
		req, err := NewCrawlRequest(u)
		if err != nil {
			return err
		}
		reqs[i] = req
	}

	crawler.Start(reqs...)

	for {
		time.Sleep(3 * time.Second)
	}

	printf("Dolan, y u do dis?\n")

	return nil
}

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

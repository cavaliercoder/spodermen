package crawler

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type Crawler struct {
	client      *http.Client
	useragent   string
	noFollow    bool
	followHosts map[string]bool
	stats       Stats
	ctx         context.Context
}

func New(options ...Option) (*Crawler, error) {
	c := &Crawler{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(time.Second * 10),
		},
		followHosts: map[string]bool{},
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Crawler) Do(req *Request) (*Response, error) {
	hreq, err := http.NewRequest("GET", req.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.ctx != nil {
		hreq = hreq.WithContext(c.ctx)
	}
	if c.useragent != "" {
		hreq.Header.Set("User-Agent", c.useragent)
	}

	start := time.Now()
	hresp, err := c.client.Do(hreq)
	if err != nil {
		return nil, err
	}

	body := ""
	if err := func() error {
		defer hresp.Body.Close()
		b, err := ioutil.ReadAll(hresp.Body) // TODO: allow cancellation
		if strings.HasPrefix(hresp.Header.Get("Content-Type"), "text/html") {
			body = string(b)
		}
		return err
	}(); err != nil {
		return nil, err
	}

	resp := &Response{
		Request:       req,
		StatusCode:    hresp.StatusCode,
		Duration:      time.Since(start),
		ContentLength: hresp.ContentLength,
		ContentType:   hresp.Header.Get("Content-Type"),
	}
	if resp.ContentType == "" {
		resp.ContentType = "-"
	} else {
		// strip after ;
		if n := strings.Index(resp.ContentType, ";"); n != -1 {
			resp.ContentType = resp.ContentType[:n]
		}
	}

	if req.Follow {
		hrefs, err := getHrefs(strings.NewReader(body))
		if err != nil {
			return nil, err
		}

		for _, href := range hrefs {
			var err error
			var furl *url.URL
			if strings.HasPrefix(href, "/") {
				furl, err = req.URL.Parse(href)
			} else {
				furl, err = url.Parse(href)
			}
			if err != nil {
				continue // ignore broken URLs
			}
			if c.followHosts[furl.Host] {
				resp.URLs = append(resp.URLs, furl.String())
			}
		}
	}
	c.stats.AddResponse(resp)
	return resp, nil
}

func (c *Crawler) Crawl(q *Queue, workers int) *Stats {
	if q.length == 0 {
		panic("cannot crawl an empty queue")
	}
	wg := &sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				req, ok := q.Dequeue()
				if !ok {
					break
				}

				resp, err := c.Do(req)
				if err != nil {
					fmt.Fprintf(os.Stdout, "%v\n", err)
					q.Done()
					continue
				}

				reqs := make([]*Request, 0, len(resp.URLs))
				for _, u := range resp.URLs {
					if i := strings.Index(u, "#"); i != -1 {
						u = u[:i] // ignore URL fragment
					}
					req, err := NewRequest(u, !c.noFollow)
					if err != nil {
						panic(err) // TODO
					}
					reqs = append(reqs, req)
				}
				go q.Enqueue(reqs...) // never block dequeuing goroutines
				fmt.Printf("%-26s %v\n", time.Now().Format("2006-01-02 15:04:05.999999"), resp)
				q.Done()
			}
		}()
		//time.Sleep(time.Millisecond * 100) // slow start
	}
	wg.Wait()
	return &c.stats
}

// getHrefs uses the HTML tokenizer to find any URLs stored in href or src
// attributes (of any element type) in the HTML document.
func getHrefs(r io.Reader) ([]string, error) {
	urls := make([]string, 0)
	z := html.NewTokenizer(r)
	for z.Err() != io.EOF {
		if tt := z.Next(); tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			for {
				k, v, ok := z.TagAttr()
				switch string(k) {
				case "href", "src":
					urls = append(urls, string(v))
				}
				if !ok {
					break
				}
			}
		}
	}
	return urls, nil
}

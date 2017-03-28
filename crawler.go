package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	// UserAgent is the UserAgent header set for all HTTP requests.
	UserAgent = "Spodermen bot/1.0"
)

type Crawler interface {
	Do(*CrawlRequest) (*CrawlResponse, error)
}

type crawler struct {
	client  *http.Client
	options *CrawlOptions
}

type CrawlOptions struct {
	NoFollow bool
}

func NewCrawler(opts *CrawlOptions) Crawler {
	return &crawler{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(time.Second * 10),
		},
		options: opts,
	}
}

func (c *crawler) Do(req *CrawlRequest) (*CrawlResponse, error) {
	resp := &CrawlResponse{
		Request: req,
	}

	hreq, err := http.NewRequest("GET", req.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("User-Agent", UserAgent)

	start := time.Now()
	hresp, err := c.client.Do(hreq)
	if err != nil {
		return nil, err
	}

	body := ""
	if err := func() error {
		defer hresp.Body.Close()
		b, err := ioutil.ReadAll(hresp.Body)
		body = string(b)
		resp.ContentLength = len(b)
		return err
	}(); err != nil {
		return nil, err
	}

	resp.Duration = time.Since(start)
	resp.StatusCode = hresp.StatusCode
	resp.ContentType = hresp.Header.Get("Content-Type")
	if resp.ContentType == "" {
		resp.ContentType = "-"
	} else {
		// strip after ;
		if n := strings.Index(resp.ContentType, ";"); n != -1 {
			resp.ContentType = resp.ContentType[:n]
		}
	}

	if !c.options.NoFollow {
		hrefs, err := getHrefs(strings.NewReader(body))
		if err != nil {
			return nil, err
		}

		for _, href := range hrefs {
			if strings.HasPrefix(href, "/") {
				uri, err := req.URL.Parse(href)
				if err != nil {
					continue // ignore broken URLs
				}

				// match only domain local URLs
				if uri.Host == req.URL.Host {
					resp.URLs = append(resp.URLs, uri.String())
				}
			}
		}
	}

	if req.Callback != nil {
		if err := req.Callback(resp); err != nil {
			return resp, fmt.Errorf("error in callback for %v: %v", req, err)
		}
	}

	return resp, nil
}

// getHrefs uses the HTML tokenizer to find any URLs stored in href or src
// attributes (of any element type) in the HTML document.
func getHrefs(r io.Reader) ([]string, error) {
	urls := make([]string, 0)
	z := html.NewTokenizer(r)
	for z.Err() != io.EOF {
		if tt := z.Next(); tt == html.StartTagToken {
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

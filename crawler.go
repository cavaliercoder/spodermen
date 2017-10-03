package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"net/url"

	"golang.org/x/net/html"
)

// UserAgent is the UserAgent header set for all HTTP requests.
var UserAgent = fmt.Sprintf("%s bot/%s", PackageName, PackageVersion)

type Crawler interface {
	Do(*CrawlRequest) (*CrawlResponse, error)
	Stats() *CrawlerStats
}

type crawler struct {
	client  *http.Client
	options *CrawlOptions
	stats   CrawlerStats
	ctx     context.Context
}

type CrawlOptions struct {
	NoFollow bool
	Hosts    map[string]bool
}

func NewCrawler(opts *CrawlOptions, ctx context.Context) Crawler {
	return &crawler{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(time.Second * 10),
		},
		options: opts,
		ctx:     ctx,
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
	hreq = hreq.WithContext(c.ctx)
	hreq.Header.Set("User-Agent", UserAgent)

	start := time.Now()
	hresp, err := c.client.Do(hreq)
	if err != nil {
		return nil, err
	}

	body := ""
	if err := func() error {
		defer hresp.Body.Close()
		b, err := ioutil.ReadAll(hresp.Body) // TODO: allow cancellation
		resp.ContentLength = int64(len(b))
		if strings.HasPrefix(hresp.Header.Get("Content-Type"), "text/html") {
			body = string(b)
		}
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
			if c.options.Hosts[furl.Host] {
				resp.URLs = append(resp.URLs, furl.String())
			}
		}
	}

	c.stats.AddResponse(resp)

	return resp, nil
}

func (c *crawler) Stats() *CrawlerStats {
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

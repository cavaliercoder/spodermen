package crawler

import (
	"net/url"
)

// A Request represents a single URL in a crawling queue.
type Request struct {
	URL    *url.URL
	Follow bool
}

func NewRequest(urlStr string, follow bool) (*Request, error) {
	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &Request{uri, follow}, nil
}

func (c *Request) String() string {
	return c.URL.String()
}

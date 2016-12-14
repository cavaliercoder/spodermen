package main

type Queue interface {
	Enqueue(*CrawlRequest)
	Dequeue() *CrawlRequest
	Close()
}

func NewQueue(n int) Queue {
	c := &queue{
		in:  make(chan *CrawlRequest, n),
		out: make(chan *CrawlRequest, n),
	}

	// filter the out queue of duplicates
	go func() {
		history := make(map[string]int, 1024)
		for req := range c.in {
			uri := req.URL.String()
			if _, ok := history[uri]; ok {
				history[uri]++
			} else {
				history[uri] = 1
				go func(req *CrawlRequest) {
					c.out <- req
				}(req)
			}
		}
	}()

	return c
}

type queue struct {
	in  chan *CrawlRequest
	out chan *CrawlRequest
}

func (c *queue) Enqueue(req *CrawlRequest) {
	go func(req *CrawlRequest) {
		c.in <- req
	}(req)
}

func (c *queue) Dequeue() *CrawlRequest {
	return <-c.out
}

func (c *queue) Close() {
	close(c.in)
	close(c.out)
}

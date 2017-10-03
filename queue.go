package main

import (
	"context"
	"sync"
)

type CrawlRequestQueue struct {
	sync.Mutex

	in        int
	out       int
	length    int
	inProcess int
	done      bool
	queue     []*CrawlRequest
	inCond    *sync.Cond
	outCond   *sync.Cond
	ctx       context.Context
	cancel    context.CancelFunc
	seen      map[string]int
}

func NewCrawlRequestQueue(size int, ctx context.Context) *CrawlRequestQueue {
	if size < 1 {
		panic("invalid queue size")
	}
	q := &CrawlRequestQueue{
		queue: make([]*CrawlRequest, size),
		seen:  make(map[string]int, 0),
	}
	q.inCond = sync.NewCond(q)
	q.outCond = sync.NewCond(q)

	q.ctx, q.cancel = context.WithCancel(ctx)
	go func() {
		// check for cancellation from caller
		<-q.ctx.Done()
		q.close()
	}()

	return q
}

func (q *CrawlRequestQueue) Enqueue(reqs ...*CrawlRequest) {
	// run in another goroutine so we don't block caller if the queue is full
	// BUG: number of goroutines can explode if the queue is under pressure -
	// can result in: net/http: request canceled (Client.Timeout exceeded while
	// reading body)
	go func() {
		q.Lock()
		defer q.Unlock()

		for _, req := range reqs {
			u := req.URL.String()
			if _, ok := q.seen[u]; ok {
				continue // skip duplicate request
			}

			for !q.done && q.length == len(q.queue) {
				q.inCond.Wait() // wait for capacity
			}
			if q.done {
				return
			}

			// enqueue request
			q.seen[u]++
			q.length++
			q.queue[q.in] = req
			q.in = (q.in + 1) % len(q.queue)
			q.outCond.Signal()
		}
	}()
}

func (q *CrawlRequestQueue) Dequeue() (*CrawlRequest, bool) {
	q.Lock()
	defer q.Unlock()
	for !q.done && q.length == 0 {
		q.outCond.Wait() // wait for work
	}
	if q.done {
		return nil, false
	}

	// dequeue request
	q.length--
	q.inProcess++
	req := q.queue[q.out]
	q.queue[q.out] = nil
	q.out = (q.out + 1) % len(q.queue)
	q.inCond.Signal()
	return req, true
}

// Done decrements the number of queue items in-process.
func (q *CrawlRequestQueue) Done() {
	q.Lock()
	defer q.Unlock()
	q.inProcess--
	if q.length == 0 && q.inProcess == 0 {
		q.Close()
	}
}

func (q *CrawlRequestQueue) Close() {
	q.cancel() // cancel via context
}

// close is called when ctx is cancelled
func (q *CrawlRequestQueue) close() {
	q.Lock()
	defer q.Unlock()
	q.done = true
	q.inCond.Broadcast()
	q.outCond.Broadcast()
}

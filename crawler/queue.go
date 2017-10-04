package crawler

import (
	"context"
	"sync"
)

type Queue struct {
	sync.Mutex

	in        int
	out       int
	length    int
	inProcess int
	done      bool
	queue     []*Request
	inCond    *sync.Cond
	outCond   *sync.Cond
	ctx       context.Context
	cancel    context.CancelFunc
	seen      map[string]int
}

func NewQueue(size int, ctx context.Context) *Queue {
	if size < 1 {
		panic("invalid queue size")
	}
	q := &Queue{
		queue: make([]*Request, size),
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

func (q *Queue) Enqueue(reqs ...*Request) {
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
}

func (q *Queue) Dequeue() (*Request, bool) {
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
func (q *Queue) Done() {
	q.Lock()
	defer q.Unlock()
	q.inProcess--
	if q.length == 0 && q.inProcess == 0 {
		q.Close()
	}
}

func (q *Queue) Close() {
	q.cancel() // cancel via context
}

// close is called when ctx is cancelled
func (q *Queue) close() {
	q.Lock()
	defer q.Unlock()
	q.done = true
	q.inCond.Broadcast()
	q.outCond.Broadcast()
}

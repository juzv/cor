package ctr

import (
	"math"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

// multi-array queue

type segment struct {
	data []interface{}
	wIdx uint32
	rIdx uint32
	next atomic.Pointer[segment]
}

type FlmaQueue struct {
	head    atomic.Pointer[segment]
	tail    atomic.Pointer[segment]
	size    uint32
	maxSpin int32
}

func newSegment(size uint32) *segment {
	return &segment{
		data: make([]interface{}, size),
	}
}

func NewFlmaQueue() *FlmaQueue {
	seg := newSegment(1024) // 每个段的大小为1024
	q := &FlmaQueue{
		size:    1024,
		maxSpin: 10,
	}
	q.head.Store(seg)
	q.tail.Store(seg)
	return q
}

func (q *FlmaQueue) backoff(spin int) int {
	spin++
	maxSpin := atomic.LoadInt32(&q.maxSpin)

	if spin >= int(maxSpin) {
		backoffDuration := time.Duration(math.Pow(2, float64(spin-int(maxSpin)))) * time.Microsecond
		jitter := time.Duration(rand.Int63n(int64(backoffDuration)))
		time.Sleep(backoffDuration + jitter)
		spin = 0
	} else {
		runtime.Gosched()
	}

	return spin
}

func (q *FlmaQueue) Put(val interface{}) bool {
	spin := 0

	for {
		tail := q.tail.Load()
		wIdx := atomic.LoadUint32(&tail.wIdx)

		if wIdx < q.size {
			if atomic.CompareAndSwapUint32(&tail.wIdx, wIdx, wIdx+1) {
				tail.data[wIdx] = val
				return true
			}
		} else {
			newSeg := newSegment(q.size)
			newSeg.data[0] = val
			newSeg.wIdx = 1
			oldTail := tail
			if oldTail.next.CompareAndSwap(nil, newSeg) {
				q.tail.CompareAndSwap(oldTail, newSeg)
				return true
			}
		}

		spin = q.backoff(spin)
	}
}

func (q *FlmaQueue) Get() interface{} {
	spin := 0

	for {
		head := q.head.Load()
		rIdx := atomic.LoadUint32(&head.rIdx)

		if rIdx < atomic.LoadUint32(&head.wIdx) {
			if atomic.CompareAndSwapUint32(&head.rIdx, rIdx, rIdx+1) {
				val := head.data[rIdx]
				return val
			}
		} else {
			next := head.next.Load()
			if next == nil {
				return nil
			}
			if q.head.CompareAndSwap(head, next) {
				// q.head.Store(next) , No need to store next again, as CAS already did it
				// Optionally, head can be set to nil or handled for GC
			}
		}

		spin = q.backoff(spin)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

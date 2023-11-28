package w

import (
	"container/ring"
	"errors"
	"sync"
)

var (
	QueueFull = errors.New("queue is full")
	QueueNull = errors.New("queue is null")
)

type RingQueue[T any] struct {
	r               *ring.Ring
	start, end, max int
	mux             sync.Mutex
}

func NewRingQueue[T any](n int) *RingQueue[T] {
	return &RingQueue[T]{
		r:     ring.New(n + 1),
		start: 0, end: 0, max: n + 1,
	}
}

func (r *RingQueue[T]) Do(f func(a any)) {
	r.mux.Lock()
	defer r.mux.Unlock()
	start := r.r.Move(r.start)
	end := r.r.Move(r.end)
	for ; start != end; start = start.Next() {
		f(start.Value.(T))
	}
}

func (r *RingQueue[T]) List() []T {
	r.mux.Lock()
	defer r.mux.Unlock()
	var items []T
	start := r.r.Move(r.start)
	end := r.r.Move(r.end)
	for ; start != end; start = start.Next() {
		items = append(items, start.Value.(T))
	}
	return items
}

func (r *RingQueue[T]) Remove(f func(a any) bool) {
	r.mux.Lock()
	defer r.mux.Unlock()

	start := r.r.Move(r.start)
	end := r.r.Move(r.end)
	isPrev := false
	for ; start != end; start = start.Next() {
		if start == r.r {
			isPrev = true
		}
		if f(start.Value.(T)) {
			end.Link(ring.New(1))
			if r.start > r.end && !isPrev {
				r.start++
				if r.start >= r.max {
					r.start -= r.max
				}
			} else {
				r.end--
				if r.end < 0 {
					r.end += r.max
				}
			}
			prev := start.Prev()
			prev.Link(start.Next())
			start = prev
		}
	}
}

func (r *RingQueue[T]) Put(val T) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.start == r.end+1 {
		return QueueFull
	} else if r.end+1 == r.max && r.start == 0 {
		return QueueFull
	}
	r.r.Move(r.end).Value = val
	r.end++
	if r.end == r.max {
		r.end = 0
	}
	return nil
}

func (r *RingQueue[T]) Get() (T, error) {
	if r.start == r.end {
		return *new(T), QueueNull
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	val := r.r.Move(r.start).Value.(T)
	r.r.Move(r.start).Value = nil
	r.start++
	if r.start == r.max {
		r.start = 0
	}
	return val, nil
}

func (r *RingQueue[T]) Len() int {
	if r.start < r.end {
		return r.end - r.start
	} else if r.start == r.end {
		return 0
	} else {
		return r.end + r.max - r.start
	}
}

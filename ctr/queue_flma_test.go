package ctr

import (
	"sync"
	"testing"
	"time"
)

func TestPutGet(t *testing.T) {
	q := NewFlmaQueue()

	// 测试单线程的入队和出队
	q.Put(1)
	q.Put(2)
	q.Put(3)

	if v := q.Get(); v != 1 {
		t.Errorf("Expected 1, got %v", v)
	}
	if v := q.Get(); v != 2 {
		t.Errorf("Expected 2, got %v", v)
	}
	if v := q.Get(); v != 3 {
		t.Errorf("Expected 3, got %v", v)
	}
	if v := q.Get(); v != nil {
		t.Errorf("Expected nil, got %v", v)
	}
}

func TestEmptyQueue(t *testing.T) {
	q := NewFlmaQueue()

	// 测试从空队列中出队
	if v := q.Get(); v != nil {
		t.Errorf("Expected nil, got %v", v)
	}
}

func TestSingleElementQueue(t *testing.T) {
	q := NewFlmaQueue()

	// 测试单个元素的入队和出队
	q.Put(42)
	if v := q.Get(); v != 42 {
		t.Errorf("Expected 42, got %v", v)
	}

	// 再次出队应得到 nil
	if v := q.Get(); v != nil {
		t.Errorf("Expected nil, got %v", v)
	}
}

func TestStress(t *testing.T) {
	q := NewFlmaQueue()
	var wg sync.WaitGroup
	num := 10000

	// 大量并发入队和出队
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				q.Put(id*1000 + j)
			}
		}(i)
	}

	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				q.Get()
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentPutGet(t *testing.T) {
	q := NewFlmaQueue()
	var wg sync.WaitGroup

	// 使用带有超时的通道
	done := make(chan struct{})

	go func() {
		defer close(done)
		// 启动多个 goroutine 进行并发入队
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					q.Put(id*100 + j)
				}
			}(i)
		}

		// 启动多个 goroutine 进行并发出队
		results := make(chan int, 1000)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					if v := q.Get(); v != nil {
						results <- v.(int)
					}
				}
			}()
		}

		wg.Wait()
		close(results)
	}()

	select {
	case <-done:
		// 正常完成
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out")
	}
}

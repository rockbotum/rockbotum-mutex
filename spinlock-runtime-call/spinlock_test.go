package mutex

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestBasicLockUnlock(t *testing.T) {
	var m Mutex

	if m.state.Load() != unlocked {
		t.Fatalf("новый мьютекс должен быть unlocked, получено %v", m.state.Load())
	}

	m.Lock()
	if m.state.Load() != locked {
		t.Errorf("после Lock() ожидалось locked, получено %v", m.state.Load())
	}

	m.Unlock()
	if m.state.Load() != unlocked {
		t.Errorf("после Unlock() ожидалось unlocked, получено %v", m.state.Load())
	}
}

func TestConcurrentAccess(t *testing.T) {
	const (
		goroutines = 8
		iterations = 1000
	)

	var (
		m       Mutex
		counter int
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.Lock()
				counter++
				m.Unlock()
			}
		}()
	}
	wg.Wait()

	if want := goroutines * iterations; counter != want {
		t.Errorf("counter = %d, ожидалось %d — взаимное исключение нарушено", counter, want)
	}
}

func TestStress(t *testing.T) {
	const (
		goroutines = 16
		iterations = 500
	)

	var (
		m          Mutex
		inside     atomic.Int32
		violations atomic.Int32
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.Lock()
				if inside.Add(1) != 1 {
					violations.Add(1)
				}
				inside.Add(-1)
				m.Unlock()
			}
		}()
	}
	wg.Wait()

	if n := violations.Load(); n != 0 {
		t.Errorf("случаев одновременного входа в критическую секцию: %d, ожидалось 0", n)
	}
}

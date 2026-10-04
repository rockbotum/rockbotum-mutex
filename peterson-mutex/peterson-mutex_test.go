package petersonmutex

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestBasicLockUnlock(t *testing.T) {
	var m Mutex

	m.Lock(0)
	if m.wants[0].Load() != locked {
		t.Errorf("после Lock(0) ожидалось wants[0] = locked, получено %v", m.wants[0].Load())
	}
	m.Unlock(0)
	if m.wants[0].Load() != unlocked {
		t.Errorf("после Unlock(0) ожидалось wants[0] = unlocked, получено %v", m.wants[0].Load())
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Реализация использует wants[2] и victim, поэтому допускает ровно
	// двух участников: индекс 0 и индекс 1 передаются явно.
	const iterations = 5000

	var (
		m       Mutex
		counter int
	)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		index := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.Lock(index)
				counter++
				m.Unlock(index)
			}
		}()
	}
	wg.Wait()

	if want := 2 * iterations; counter != want {
		t.Errorf("counter = %d, ожидалось %d — взаимное исключение нарушено", counter, want)
	}
}

func TestStress(t *testing.T) {
	const iterations = 5000

	var (
		m          Mutex
		inside     atomic.Int32
		violations atomic.Int32
	)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		index := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.Lock(index)
				if inside.Add(1) != 1 {
					violations.Add(1)
				}
				inside.Add(-1)
				m.Unlock(index)
			}
		}()
	}
	wg.Wait()

	if n := violations.Load(); n != 0 {
		t.Errorf("случаев одновременного входа в критическую секцию: %d, ожидалось 0", n)
	}
}

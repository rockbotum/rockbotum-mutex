package ticketlock

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestBasicLockUnlock(t *testing.T) {
	var m Mutex

	if m.ownerTicket.Load() != 0 || m.nextFreeTicket.Load() != 0 {
		t.Fatalf("новый мьютекс должен иметь нулевые счётчики, получено owner=%d next=%d",
			m.ownerTicket.Load(), m.nextFreeTicket.Load())
	}

	m.Lock()
	if m.ownerTicket.Load() != 0 {
		t.Errorf("после Lock() ожидался билет 0, получено %d", m.ownerTicket.Load())
	}

	m.Unlock()
	if m.ownerTicket.Load() != 1 {
		t.Errorf("после Unlock() ожидался билет 1, получено %d", m.ownerTicket.Load())
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

// TestFIFOOrder проверяет главное свойство ticket lock: порядок выдачи
// билетов совпадает с порядком входа в критическую секцию.
func TestFIFOOrder(t *testing.T) {
	const participants = 8

	var (
		m     Mutex
		order []int
		wg    sync.WaitGroup
	)

	wg.Add(participants)
	for i := 0; i < participants; i++ {
		go func() {
			defer wg.Done()
			m.Lock()
			// Внутри критической секции ownerTicket равен ровно билету
			// текущей горутины, поэтому чтения строго возрастают по порядку.
			order = append(order, int(m.ownerTicket.Load()))
			m.Unlock()
		}()
	}
	wg.Wait()

	if len(order) != participants {
		t.Fatalf("входов в критическую секцию: %d, ожидалось %d", len(order), participants)
	}
	for i, got := range order {
		if got != i {
			t.Errorf("на позиции %d ожидался билет %d, получен %d — порядок не FIFO", i, i, got)
		}
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

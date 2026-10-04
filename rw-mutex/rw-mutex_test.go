package rwmutex

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBasicLockUnlock(t *testing.T) {
	rw := NewRWMutex()

	rw.Lock()
	if !rw.hasWriter {
		t.Error("после Lock() ожидалось hasWriter = true")
	}
	if rw.readers != 0 {
		t.Errorf("после Lock() ожидалось readers = 0, получено %d", rw.readers)
	}
	rw.Unlock()
	if rw.hasWriter {
		t.Error("после Unlock() ожидалось hasWriter = false")
	}

	rw.RLock()
	if rw.readers != 1 {
		t.Errorf("после RLock() ожидалось readers = 1, получено %d", rw.readers)
	}
	rw.RUnlock()
	if rw.readers != 0 {
		t.Errorf("после RUnlock() ожидалось readers = 0, получено %d", rw.readers)
	}
}

// TestReadConcurrency проверяет, что читатели действительно работают
// параллельно, а не выстраиваются в очередь.
func TestReadConcurrency(t *testing.T) {
	const readers = 4

	rw := NewRWMutex()

	var (
		inside  atomic.Int32
		release = make(chan struct{})
		wg      sync.WaitGroup
	)

	wg.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			rw.RLock()
			inside.Add(1)
			<-release
			rw.RUnlock()
		}()
	}

	deadline := time.Now().Add(10 * time.Second)
	for inside.Load() < readers && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	concurrent := inside.Load()
	close(release)
	wg.Wait()

	if concurrent != readers {
		t.Errorf("одновременно внутри RLock: %d, ожидалось %d — читатели сериализуются", concurrent, readers)
	}
}

// TestWriterExcludesReaders проверяет, что пока writer удерживает лок,
// новые читатели не могут войти в критическую секцию.
func TestWriterExcludesReaders(t *testing.T) {
	rw := NewRWMutex()

	var readerEntered atomic.Bool

	rw.Lock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		rw.RLock()
		readerEntered.Store(true)
		rw.RUnlock()
	}()

	select {
	case <-done:
		t.Error("RLock() прошёл, пока writer удерживал лок")
	case <-time.After(100 * time.Millisecond):
	}

	rw.Unlock()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("читатель не смог получить лок после Unlock()")
	}

	if !readerEntered.Load() {
		t.Error("читатель не вошёл в критическую секцию после Unlock()")
	}
}

func TestConcurrentAccess(t *testing.T) {
	const (
		readers    = 4
		writers    = 2
		iterations = 250
	)

	rw := NewRWMutex()

	// writeCounter трогают только writer-ы. Они взаимоисключающи и не
	// пересекаются с читателями, поэтому потерянное обновление здесь
	// означала бы нарушение эксклюзивности.
	var writeCounter int
	// operations считает все операции: читатели не исключают друг друга,
	// поэтому этот счётчик обязан быть атомарным.
	var operations atomic.Int64

	var wg sync.WaitGroup
	wg.Add(readers + writers)

	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rw.RLock()
				operations.Add(1)
				rw.RUnlock()
			}
		}()
	}
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rw.Lock()
				writeCounter++
				operations.Add(1)
				rw.Unlock()
			}
		}()
	}
	wg.Wait()

	if want := int64((readers + writers) * iterations); operations.Load() != want {
		t.Errorf("операций: %d, ожидалось %d", operations.Load(), want)
	}
	if want := writers * iterations; writeCounter != want {
		t.Errorf("writeCounter = %d, ожидалось %d — lost update в секции writer", writeCounter, want)
	}
}

func TestStress(t *testing.T) {
	const (
		readers    = 6
		writers    = 2
		iterations = 200
	)

	rw := NewRWMutex()

	var (
		writeCounter int
		operations   atomic.Int64
		readersIn    atomic.Int32
		writerIn     atomic.Bool
		violations   atomic.Int32
	)

	var wg sync.WaitGroup
	wg.Add(readers + writers)

	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rw.RLock()
				if writerIn.Load() {
					violations.Add(1)
				}
				readersIn.Add(1)
				operations.Add(1)
				readersIn.Add(-1)
				rw.RUnlock()
			}
		}()
	}
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rw.Lock()
				// Два writer внутри секции либо reader рядом с writer —
				// оба случая означают сломанную синхронизацию.
				if !writerIn.CompareAndSwap(false, true) {
					violations.Add(1)
				}
				if readersIn.Load() != 0 {
					violations.Add(1)
				}
				writeCounter++
				operations.Add(1)
				writerIn.Store(false)
				rw.Unlock()
			}
		}()
	}
	wg.Wait()

	if n := violations.Load(); n != 0 {
		t.Errorf("нарушений эксклюзивности: %d, ожидалось 0", n)
	}
	if want := int64((readers + writers) * iterations); operations.Load() != want {
		t.Errorf("операций: %d, ожидалось %d", operations.Load(), want)
	}
	if want := writers * iterations; writeCounter != want {
		t.Errorf("writeCounter = %d, ожидалось %d — lost update в секции writer", writeCounter, want)
	}
}

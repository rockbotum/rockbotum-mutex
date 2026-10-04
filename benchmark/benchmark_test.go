// Package benchmark сравнивает реализации мьютексов из репозитория
// с эталонным sync.Mutex из стандартной библиотеки.
//
// Запуск:
//
//	go test ./benchmark -bench=. -benchmem
package benchmark

import (
	"sync"
	"testing"

	mutexaction "rockbotummutex/mutex-with-action"
	petersonmutex "rockbotummutex/peterson-mutex"
	retrymutex "rockbotummutex/retry-mutex"
	rwmutex "rockbotummutex/rw-mutex"
	spinlock "rockbotummutex/spinlock"
	spinlockrt "rockbotummutex/spinlock-runtime-call"
	ticketlock "rockbotummutex/ticket-lock"
)

// locker — общий интерфейс для реализаций с обычной парой Lock/Unlock.
type locker interface {
	Lock()
	Unlock()
}

type implementation struct {
	name string
	new  func() locker
}

// implementations — всё, что можно сравнить на одинаковой нагрузке.
var implementations = []implementation{
	{"sync.Mutex", func() locker { return &sync.Mutex{} }},
	{"spinlock", func() locker { return &spinlock.Mutex{} }},
	{"spinlock-runtime-call", func() locker { return &spinlockrt.Mutex{} }},
	{"retry-mutex", func() locker { return &retrymutex.Mutex{} }},
	{"ticket-lock", func() locker { return &ticketlock.Mutex{} }},
}

// BenchmarkUncontended измеряет стоимость Lock/Unlock без конкуренции.
func BenchmarkUncontended(b *testing.B) {
	for _, impl := range implementations {
		b.Run(impl.name, func(b *testing.B) {
			m := impl.new()
			b.ReportAllocs()
			for b.Loop() {
				m.Lock()
				m.Unlock()
			}
		})
	}
}

// BenchmarkContended измеряет поведение под конкуренцией: горутины
// просыпаются и contend-ятся на одном локе.
func BenchmarkContended(b *testing.B) {
	for _, impl := range implementations {
		b.Run(impl.name, func(b *testing.B) {
			m := impl.new()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					m.Lock()
					m.Unlock()
				}
			})
		})
	}
}

func BenchmarkRWMutexRead(b *testing.B) {
	b.Run("sync.RWMutex", func(b *testing.B) {
		m := &sync.RWMutex{}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.RLock()
				m.RUnlock()
			}
		})
	})
	b.Run("rw-mutex", func(b *testing.B) {
		m := rwmutex.NewRWMutex()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.RLock()
				m.RUnlock()
			}
		})
	})
}

func BenchmarkRWMutexWrite(b *testing.B) {
	b.Run("sync.RWMutex", func(b *testing.B) {
		m := &sync.RWMutex{}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Lock()
				m.Unlock()
			}
		})
	})
	b.Run("rw-mutex", func(b *testing.B) {
		m := rwmutex.NewRWMutex()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Lock()
				m.Unlock()
			}
		})
	})
}

// BenchmarkMutexWithAction измеряет вариант, который выполняет действие
// под локом, поэтому Unlock() вызывается явно после Lock().
func BenchmarkMutexWithAction(b *testing.B) {
	b.Run("uncontended", func(b *testing.B) {
		m := &mutexaction.Mutex{}
		b.ReportAllocs()
		for b.Loop() {
			m.Lock(func() any { return nil })
			m.Unlock()
		}
	})
	b.Run("contended", func(b *testing.B) {
		m := &mutexaction.Mutex{}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Lock(func() any { return nil })
				m.Unlock()
			}
		})
	})
}

// BenchmarkPetersonMutex вынесен отдельно: реализация допускает ровно двух
// участников с явно переданными индексами, поэтому b.RunParallel к ней
// неприменим, а сравнение с остальными на разном числе горутин было бы
// некорректным.
func BenchmarkPetersonMutex(b *testing.B) {
	const participants = 2

	b.Run("contended_2_goroutines", func(b *testing.B) {
		m := &petersonmutex.Mutex{}
		b.ReportAllocs()

		// b.N читается до старта горутин: обращаться к нему из горутин
		// нельзя, значение меняется по ходу прогона.
		perGoroutine := b.N / participants

		var wg sync.WaitGroup
		wg.Add(participants)
		for i := 0; i < participants; i++ {
			index := i
			go func() {
				defer wg.Done()
				for j := 0; j < perGoroutine; j++ {
					m.Lock(index)
					m.Unlock(index)
				}
			}()
		}
		wg.Wait()
	})
}

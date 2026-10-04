package petersonmutex

import (
	"runtime"
	"sync/atomic"
)

const (
	unlocked = false
	locked   = true
)

type Mutex struct {
	wants  [2]atomic.Bool
	owner  int
	victim atomic.Int32
}

func (m *Mutex) Lock(index int) {
	m.wants[index].Store(locked)
	m.victim.Store(int32(index))

	otherIndex := 1 - index
	for m.wants[otherIndex].Load() == locked && m.victim.Load() == int32(index) {
		runtime.Gosched() // Yield the processor to allow other goroutines to run
	}
}

func (m *Mutex) Unlock(index int) {
	m.wants[index].Store(unlocked)
}

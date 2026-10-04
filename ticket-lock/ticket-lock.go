package ticketlock

import (
	"runtime"
	"sync/atomic"
)

type Mutex struct {
	ownerTicket    atomic.Int64
	nextFreeTicket atomic.Int64
}

func (m *Mutex) Lock() {
	ticket := m.nextFreeTicket.Add(1)
	for m.ownerTicket.Load() != ticket-1 {
		runtime.Gosched() // Yield the processor to allow other goroutines to run
	}
}

func (m *Mutex) Unlock() {
	m.ownerTicket.Add(1)
}

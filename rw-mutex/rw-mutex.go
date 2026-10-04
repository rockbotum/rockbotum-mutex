package rwmutex

import (
	"sync"
)

type RWMutex struct {
	notifier  *sync.Cond
	mutex     *sync.Mutex
	readers   int
	hasWriter bool
}

func NewRWMutex() *RWMutex {
	var mutex sync.Mutex
	notifier := sync.NewCond(&mutex)

	return &RWMutex{
		notifier: notifier,
		mutex:    &mutex,
	}
}

func (rw *RWMutex) Lock() {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	for rw.hasWriter {
		rw.notifier.Wait()
	}

	rw.hasWriter = true
	for rw.readers > 0 {
		rw.notifier.Wait()
	}
}

func (rw *RWMutex) Unlock() {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	rw.hasWriter = false
	rw.notifier.Broadcast()
}

func (rw *RWMutex) RLock() {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	for rw.hasWriter {
		rw.notifier.Wait()
	}

	rw.readers++
}

func (rw *RWMutex) RUnlock() {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	rw.readers--
	if rw.readers == 0 {
		rw.notifier.Broadcast()
	}
}

package servertools

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// Closable offers a way to cleanly shut down a running goroutine.
type Closable struct {
	doneChan     chan struct{}
	doneWg       *sync.WaitGroup
	isClosed     bool
	closingMutex *sync.Mutex
}

// MakeClosable constructs a new closable struct
func MakeClosable() Closable {
	return Closable{
		make(chan struct{}),
		new(sync.WaitGroup),
		false,
		new(sync.Mutex),
	}
}

// ClosableAdd should be combined with 'delta' calls to closableDone()
func (closable *Closable) ClosableAdd(delta int) {
	closable.closingMutex.Lock()
	defer closable.closingMutex.Unlock()

	log.Debugf("Closable: doneWg.Add(%d) ok", delta)
	closable.doneWg.Add(delta)
}

// ClosableDone marks one "thing" as "done"
func (closable *Closable) ClosableDone() {
	closable.closingMutex.Lock()
	defer closable.closingMutex.Unlock()

	log.Debug("Closable: doneWg.Done() ok")
	closable.doneWg.Done()
}

// closableMaybeClose only closes the channel if it wasn't closed yet.
func (closable *Closable) closableMaybeClose() {
	closable.closingMutex.Lock()
	defer closable.closingMutex.Unlock()

	if !closable.isClosed {
		closable.isClosed = true
		close(closable.doneChan)
	}
}

// ClosableCloseAndWait marks the goroutine as "done",
// and waits for all things added with closableAdd() to be "done" too.
func (closable *Closable) ClosableCloseAndWait() {
	closable.closableMaybeClose()
	log.Debug("Closable: waiting for shutdown to finish.")
	closable.doneWg.Wait()
}

// ClosableCloseNotWait marks the goroutine as "done",
// but does not waits for all things added with closableAdd() to be "done" too.
func (closable *Closable) ClosableCloseNotWait() {
	closable.closableMaybeClose()
	log.Debug("Closable: marking as closed but NOT waiting shutdown to finish.")
}

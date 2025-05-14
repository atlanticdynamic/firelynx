package testutil

import (
	"bytes"
	"sync"
)

// ThreadSafeBuffer is a thread-safe wrapper around bytes.Buffer
type ThreadSafeBuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write implements io.Writer
func (b *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

// String returns the accumulated buffer as a string
func (b *ThreadSafeBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

// Reset resets the buffer to be empty
func (b *ThreadSafeBuffer) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.buffer.Reset()
}

package testutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomPort(t *testing.T) {
	port := GetRandomPort(t)
	assert.Greater(t, port, 0)
	assert.Less(t, port, 65536)
}

func TestGetRandomPortUnique(t *testing.T) {
	ports := make(map[int]bool)
	for range 10 {
		port := GetRandomPort(t)
		assert.False(t, ports[port], "Port %d was already used", port)
		ports[port] = true
	}
}

func TestGetRandomPortConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	portChan := make(chan int, 20)

	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			port := GetRandomPort(t)
			portChan <- port
		}()
	}

	wg.Wait()
	close(portChan)

	ports := make(map[int]bool)
	for port := range portChan {
		assert.False(t, ports[port], "Port %d was already used", port)
		ports[port] = true
	}
}

func TestGetRandomListeningPort(t *testing.T) {
	addr := GetRandomListeningPort(t)
	assert.Contains(t, addr, "localhost:")
	assert.Greater(t, len(addr), len("localhost:"))
}

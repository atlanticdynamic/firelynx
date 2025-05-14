package testutil

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

// reduce the chance of port conflicts
var (
	portMutex = &sync.Mutex{}
	usedPorts = make(map[int]struct{})
)

// GetRandomPort finds an available port for the test by binding to port 0
func GetRandomPort(t *testing.T) int {
	t.Helper()
	portMutex.Lock()
	defer portMutex.Unlock()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get random port: %v", err)
	}

	err = listener.Close()
	if err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	p := addr.Port
	// Check if the port is already used
	if _, ok := usedPorts[p]; ok {
		return GetRandomPort(t)
	}
	usedPorts[p] = struct{}{}
	return p
}

// GetRandomListeningPort finds an available port for the test by binding to port 0
func GetRandomListeningPort(t *testing.T) string {
	t.Helper()
	p := GetRandomPort(t)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		return GetRandomListeningPort(t)
	}
	err = listener.Close()
	if err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	return fmt.Sprintf("localhost:%d", p)
}

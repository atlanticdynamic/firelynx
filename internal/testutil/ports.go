package testutil

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

var (
	portMutex = &sync.Mutex{}
	usedPorts = make(map[int]struct{})
)

func GetRandomPort(t *testing.T) int {
	t.Helper()
	portMutex.Lock()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		portMutex.Unlock()
		t.Fatalf("Failed to get random port: %v", err)
	}

	err = listener.Close()
	if err != nil {
		portMutex.Unlock()
		t.Fatalf("Failed to close listener: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	p := addr.Port
	if _, ok := usedPorts[p]; ok {
		portMutex.Unlock()
		return GetRandomPort(t)
	}
	usedPorts[p] = struct{}{}
	portMutex.Unlock()
	return p
}

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

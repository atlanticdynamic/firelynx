package http

import (
	"net"
	"testing"
)

// GetRandomPort finds an available port for the test by binding to port 0
func GetRandomPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get random port: %v", err)
	}

	err = listener.Close()
	if err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

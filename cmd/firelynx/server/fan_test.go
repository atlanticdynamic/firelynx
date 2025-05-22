package server

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfigProvider is a test implementation of ConfigChannelProvider
type mockConfigProvider struct {
	ch      chan *transaction.ConfigTransaction
	closeCh bool
}

func newMockConfigProvider(buffer int) *mockConfigProvider {
	return &mockConfigProvider{
		ch: make(chan *transaction.ConfigTransaction, buffer),
	}
}

func (m *mockConfigProvider) GetConfigChan() <-chan *transaction.ConfigTransaction {
	return m.ch
}

func (m *mockConfigProvider) send(tx *transaction.ConfigTransaction) {
	m.ch <- tx
}

func (m *mockConfigProvider) close() {
	if !m.closeCh {
		close(m.ch)
		m.closeCh = true
	}
}

func TestFanInConfigProvider_BasicOperations(t *testing.T) {
	t.Parallel()

	t.Run("single provider", func(t *testing.T) {
		ctx := t.Context()

		provider := newMockConfigProvider(1)
		fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
		require.NoError(t, err)

		outCh := fanIn.GetConfigChan()

		// Create a test transaction
		tx := &transaction.ConfigTransaction{}

		// Send transaction through provider
		provider.send(tx)

		// Should receive the transaction
		select {
		case received := <-outCh:
			assert.Equal(t, tx, received)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for transaction")
		}
	})

	t.Run("multiple providers", func(t *testing.T) {
		ctx := t.Context()

		provider1 := newMockConfigProvider(1)
		provider2 := newMockConfigProvider(1)
		provider3 := newMockConfigProvider(1)

		fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{
			provider1,
			provider2,
			provider3,
		})
		require.NoError(t, err)

		outCh := fanIn.GetConfigChan()

		// Create test transactions
		tx1 := &transaction.ConfigTransaction{}
		tx2 := &transaction.ConfigTransaction{}
		tx3 := &transaction.ConfigTransaction{}

		// Send transactions through different providers
		go provider1.send(tx1)
		go provider2.send(tx2)
		go provider3.send(tx3)

		// Should receive all transactions (order may vary)
		received := make([]*transaction.ConfigTransaction, 0, 3)
		for i := 0; i < 3; i++ {
			select {
			case tx := <-outCh:
				received = append(received, tx)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("timeout waiting for transaction %d", i+1)
			}
		}

		// Verify we got all transactions
		assert.Len(t, received, 3)
		assert.Contains(t, received, tx1)
		assert.Contains(t, received, tx2)
		assert.Contains(t, received, tx3)
	})
}

func TestFanInConfigProvider_ChannelClosing(t *testing.T) {
	t.Parallel()

	t.Run("one provider closes", func(t *testing.T) {
		ctx := t.Context()

		provider1 := newMockConfigProvider(1)
		provider2 := newMockConfigProvider(1)

		fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{
			provider1,
			provider2,
		})
		require.NoError(t, err)

		outCh := fanIn.GetConfigChan()

		// Close one provider
		provider1.close()

		// Send transaction through the other provider
		tx := &transaction.ConfigTransaction{}
		provider2.send(tx)

		// Should still receive the transaction
		select {
		case received := <-outCh:
			assert.Equal(t, tx, received)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for transaction")
		}
	})
}

func TestFanInConfigProvider_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	provider := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
	require.NoError(t, err)

	outCh := fanIn.GetConfigChan()

	// Cancel context
	cancel()

	// Output channel should eventually close
	assert.Eventually(t, func() bool {
		select {
		case _, ok := <-outCh:
			return !ok // channel closed
		default:
			return false
		}
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestFanInConfigProvider_BackpressureHandling(t *testing.T) {
	ctx := t.Context()

	provider := newMockConfigProvider(10) // Larger buffer
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
	require.NoError(t, err)

	outCh := fanIn.GetConfigChan()

	// Send multiple transactions quickly
	for i := 0; i < 5; i++ {
		provider.send(&transaction.ConfigTransaction{})
	}

	// Read them all
	received := 0
	timeout := time.After(500 * time.Millisecond)
	for received < 5 {
		select {
		case <-outCh:
			received++
		case <-timeout:
			t.Fatalf("timeout waiting for transactions, received %d/5", received)
		}
	}

	assert.Equal(t, 5, received)
}

func TestFanInConfigProvider_EmptyProviders(t *testing.T) {
	ctx := t.Context()

	// Create fan-in with no providers should return an error
	_, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoProviders)
}

func TestFanInConfigProvider_RapidContextCancellation(t *testing.T) {
	// Test that fan-in handles immediate context cancellation gracefully
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	provider := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
	require.NoError(t, err)

	outCh := fanIn.GetConfigChan()

	// Channel should close quickly
	select {
	case _, ok := <-outCh:
		if ok {
			// Might receive a value if provider had one ready
			// Try again to ensure channel closes
			select {
			case _, ok := <-outCh:
				require.False(t, ok, "channel should be closed")
			case <-time.After(100 * time.Millisecond):
				t.Fatal("channel did not close after context cancellation")
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for channel to close")
	}
}

func TestFanInConfigProvider_ConcurrentSends(t *testing.T) {
	ctx := t.Context()

	numProviders := 10
	providers := make([]txmgr.ConfigChannelProvider, numProviders)
	mockProviders := make([]*mockConfigProvider, numProviders)

	for i := 0; i < numProviders; i++ {
		mockProviders[i] = newMockConfigProvider(1)
		providers[i] = mockProviders[i]
	}

	fanIn, err := newFanInConfigProvider(ctx, providers)
	require.NoError(t, err)
	outCh := fanIn.GetConfigChan()

	// Send transactions concurrently from all providers
	for i := 0; i < numProviders; i++ {
		go func(idx int) {
			mockProviders[idx].send(&transaction.ConfigTransaction{})
		}(i)
	}

	// Receive all transactions
	received := 0
	timeout := time.After(1 * time.Second)
	for received < numProviders {
		select {
		case <-outCh:
			received++
		case <-timeout:
			t.Fatalf("timeout waiting for transactions, received %d/%d", received, numProviders)
		}
	}

	assert.Equal(t, numProviders, received)
}

func TestFanInConfigProvider_AllProvidersClose(t *testing.T) {
	ctx := t.Context()

	provider1 := newMockConfigProvider(1)
	provider2 := newMockConfigProvider(1)
	provider3 := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{
		provider1,
		provider2,
		provider3,
	})
	require.NoError(t, err)
	outCh := fanIn.GetConfigChan()

	provider1.send(&transaction.ConfigTransaction{})
	provider2.send(&transaction.ConfigTransaction{})
	provider3.send(&transaction.ConfigTransaction{})

	for i := 0; i < 3; i++ {
		select {
		case <-outCh:
			// ok
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for transaction %d", i+1)
		}
	}
	provider1.close()
	provider2.close()
	provider3.close()
	assert.Eventually(t, func() bool {
		_, ok := <-outCh
		return !ok
	}, 500*time.Millisecond, 10*time.Millisecond, "output channel should close after all providers close")
}

func TestFanInConfigProvider_CloseMethod(t *testing.T) {
	ctx := t.Context()
	provider := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
	require.NoError(t, err)
	outCh := fanIn.GetConfigChan()
	fanIn.Close()
	assert.Eventually(t, func() bool {
		_, ok := <-outCh
		return !ok
	}, 100*time.Millisecond, 10*time.Millisecond, "output channel should close after Close() is called")
}

func TestFanInConfigProvider_CloseBeforeRead(t *testing.T) {
	ctx := t.Context()
	provider := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
	require.NoError(t, err)
	tx := &transaction.ConfigTransaction{}
	provider.send(tx)
	fanIn.Close()
	outCh := fanIn.GetConfigChan()
	select {
	case received, ok := <-outCh:
		if ok {
			assert.Equal(t, tx, received, "should receive the buffered transaction")
			_, ok = <-outCh
			assert.False(t, ok, "channel should be closed after draining")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for channel state")
	}
}

func TestFanInConfigProvider_MixedCloseScenarios(t *testing.T) {
	ctx := t.Context()
	provider1 := newMockConfigProvider(1)
	provider2 := newMockConfigProvider(1)
	provider3 := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{
		provider1,
		provider2,
		provider3,
	})
	require.NoError(t, err)
	outCh := fanIn.GetConfigChan()
	provider1.close()
	tx2 := &transaction.ConfigTransaction{}
	provider2.send(tx2)
	select {
	case received := <-outCh:
		assert.Equal(t, tx2, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for transaction")
	}
	provider2.close()
	tx3 := &transaction.ConfigTransaction{}
	provider3.send(tx3)
	select {
	case received := <-outCh:
		assert.Equal(t, tx3, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for transaction")
	}
	provider3.close()
	assert.Eventually(t, func() bool {
		_, ok := <-outCh
		return !ok
	}, 100*time.Millisecond, 10*time.Millisecond, "output channel should close after all providers close")
}

func TestFanInConfigProvider_GetConfigChanIdempotent(t *testing.T) {
	ctx := t.Context()
	provider := newMockConfigProvider(1)
	fanIn, err := newFanInConfigProvider(ctx, []txmgr.ConfigChannelProvider{provider})
	require.NoError(t, err)
	ch1 := fanIn.GetConfigChan()
	ch2 := fanIn.GetConfigChan()
	ch3 := fanIn.GetConfigChan()
	assert.Equal(t, ch1, ch2, "GetConfigChan should return the same channel")
	assert.Equal(t, ch2, ch3, "GetConfigChan should return the same channel")
	tx := &transaction.ConfigTransaction{}
	provider.send(tx)
	select {
	case received := <-ch1:
		assert.Equal(t, tx, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for transaction")
	}
}

func TestFanInConfigProvider_BufferSizeConfiguration(t *testing.T) {
	ctx := t.Context()
	numProviders := 5
	providers := make([]txmgr.ConfigChannelProvider, numProviders)
	mockProviders := make([]*mockConfigProvider, numProviders)
	for i := 0; i < numProviders; i++ {
		mockProviders[i] = newMockConfigProvider(1)
		providers[i] = mockProviders[i]
	}

	fanIn, err := newFanInConfigProvider(ctx, providers)
	require.NoError(t, err)
	outCh := fanIn.GetConfigChan()
	var wg sync.WaitGroup
	for i := 0; i < numProviders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 2; j++ {
				mockProviders[idx].send(&transaction.ConfigTransaction{})
			}
		}(i)
	}
	received := 0
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	timeout := time.After(1 * time.Second)
	for received < numProviders*2 {
		select {
		case <-outCh:
			received++
		case <-done:
			// All senders finished
		case <-timeout:
			t.Fatalf("timeout waiting for transactions, received %d/%d", received, numProviders*2)
		}
	}
	assert.Equal(t, numProviders*2, received)
}

func TestFanInConfigProvider_NoGoroutineLeaks(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(t.Context())
	numProviders := 10
	providers := make([]txmgr.ConfigChannelProvider, numProviders)
	mockProviders := make([]*mockConfigProvider, numProviders)
	for i := 0; i < numProviders; i++ {
		mockProviders[i] = newMockConfigProvider(1)
		providers[i] = mockProviders[i]
	}
	fanIn, err := newFanInConfigProvider(ctx, providers)
	require.NoError(t, err)
	_ = fanIn.GetConfigChan()
	time.Sleep(50 * time.Millisecond)
	for i := 0; i < numProviders; i++ {
		mockProviders[i].close()
	}
	cancel()
	assert.Eventually(t, func() bool {
		return runtime.NumGoroutine() <= initialGoroutines+1
	}, 1*time.Second, 50*time.Millisecond, "goroutines should be cleaned up")
}

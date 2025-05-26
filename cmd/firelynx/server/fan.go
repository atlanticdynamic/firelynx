package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
)

// interface guard to ensure fanInConfigProvider implements ConfigChannelProvider
var _ txmgr.ConfigChannelProvider = (*fanInConfigProvider)(nil)

// ErrNoProviders is returned when attempting to create a fan-in with no providers
var ErrNoProviders = errors.New("at least one provider is required")

// fanInConfigProvider combines multiple ConfigChannelProviders into one.
// It follows Go's channel ownership pattern: each source provider is responsible
// for closing its own channel, and the fan-in will close its output channel
// only after all source channels have been closed.
type fanInConfigProvider struct {
	ctx        context.Context
	providers  []txmgr.ConfigChannelProvider
	bufferSize int

	once    sync.Once
	outChan <-chan *transaction.ConfigTransaction
	cancel  context.CancelFunc
}

// fanInOrDirect is a helper function that returns a ConfigChannelProvider
// that either fans in multiple providers or returns a single provider directly.
func fanInOrDirect(
	ctx context.Context,
	providers []txmgr.ConfigChannelProvider,
) (txmgr.ConfigChannelProvider, error) {
	if len(providers) == 1 {
		return providers[0], nil
	}
	return newFanInConfigProvider(ctx, providers)
}

// newFanInConfigProvider creates a new fan-in config provider that merges
// configuration transactions from multiple sources into a single channel.
//
// Buffer size defaults to the number of providers to handle the worst case
// where all providers emit simultaneously without blocking.
//
// The fan-in respects Go's channel ownership: it doesn't close source channels
// but detects when they're closed by their producers.
func newFanInConfigProvider(
	ctx context.Context,
	providers []txmgr.ConfigChannelProvider,
) (*fanInConfigProvider, error) {
	if len(providers) == 0 {
		return nil, ErrNoProviders
	}

	// Use unbuffered channels for proper backpressure
	bufferSize := 0

	// Create a cancellable context for this fan-in
	fanInCtx, cancel := context.WithCancel(ctx)

	return &fanInConfigProvider{
		ctx:        fanInCtx,
		providers:  providers,
		bufferSize: bufferSize,
		cancel:     cancel,
	}, nil
}

// GetConfigChan returns a channel that receives configuration transactions
// from all registered providers. The channel is closed when either:
// - All source channels are closed by their producers, OR
// - The context is cancelled
//
// This method is safe to call multiple times and will return the same channel.
// The fan-in does not close source channels - it expects each provider to
// close its own channel when done producing (following Go conventions).
func (f *fanInConfigProvider) GetConfigChan() <-chan *transaction.ConfigTransaction {
	f.once.Do(func() {
		f.outChan = f.startFanIn()
	})
	return f.outChan
}

func (f *fanInConfigProvider) startFanIn() <-chan *transaction.ConfigTransaction {
	out := make(chan *transaction.ConfigTransaction)

	// Track active goroutines
	activeSources := int32(len(f.providers))
	var wg sync.WaitGroup

	for _, provider := range f.providers {
		wg.Add(1)
		go func(p txmgr.ConfigChannelProvider) {
			defer wg.Done()
			ch := p.GetConfigChan()
			defer func() {
				// Decrement active sources and close output if we're the last one
				if atomic.AddInt32(&activeSources, -1) == 0 {
					close(out)
				}
			}()

			for {
				select {
				case tx, ok := <-ch:
					if !ok {
						// Source channel closed, exit this goroutine
						return
					}

					// Try to send, but respect context cancellation
					select {
					case out <- tx:
					case <-f.ctx.Done():
						return
					}

				case <-f.ctx.Done():
					return
				}
			}
		}(provider)
	}

	// Handle context cancellation
	go func() {
		<-f.ctx.Done()

		// Wait for all goroutines to finish or force close
		// Give goroutines a chance to exit cleanly
		wg.Wait()

		// If somehow the channel isn't closed yet, close it
		if atomic.LoadInt32(&activeSources) > 0 {
			close(out)
		}
	}()

	return out
}

// Close cancels the fan-in's context, signaling all goroutines to exit.
// Note: This does NOT close the source channels - each provider is responsible
// for closing its own channel. This method stops the fan-in from reading.
func (f *fanInConfigProvider) Close() {
	f.cancel()
}

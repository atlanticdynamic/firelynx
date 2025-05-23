package txmgr

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

// ConfigChannelProvider defines the interface for getting a channel of validated config transactions
type ConfigChannelProvider interface {
	GetConfigChan() <-chan *transaction.ConfigTransaction
}

type SagaProcessor interface {
	ProcessTransaction(ctx context.Context, tx *transaction.ConfigTransaction) error
	AddToStorage(tx *transaction.ConfigTransaction) error
}

package txmgr

import (
	"context"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
)

type SagaProcessor interface {
	ProcessTransaction(ctx context.Context, tx *transaction.ConfigTransaction) error
	AddToStorage(tx *transaction.ConfigTransaction) error
}

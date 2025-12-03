package reconciler

import (
	"context"
	"time"
)

func (r *reconcilerImpl) StartPeriodicReconciliation(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do initial reconciliation immediately
	r.reconcileAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reconcileAll(ctx)
		}
	}
}

// WaitForFirstReconciliation waits for the first reconciliation to complete.
// This is useful for testing to ensure the reconciler is ready.
func (r *reconcilerImpl) WaitForFirstReconciliation(ctx context.Context) error {
	select {
	case <-r.firstReconcileCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}


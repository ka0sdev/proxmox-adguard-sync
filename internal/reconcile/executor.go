package reconcile

import (
	"context"
	"fmt"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/adguard"
)

type RewriteClient interface {
	AddRewrite(
		ctx context.Context,
		rewrite adguard.Rewrite,
	) error

	UpdateRewrite(
		ctx context.Context,
		current adguard.Rewrite,
		desired adguard.Rewrite,
	) error

	DeleteRewrite(
		ctx context.Context,
		rewrite adguard.Rewrite,
	) error
}

type ExecutionResult struct {
	Added   int
	Updated int
	Deleted int
}

func Execute(
	ctx context.Context,
	client RewriteClient,
	plan Plan,
) (ExecutionResult, error) {
	result := ExecutionResult{}

	for _, rewrite := range plan.Add {
		if err := client.AddRewrite(ctx, rewrite); err != nil {
			return result, fmt.Errorf(
				"execute rewrite addition for %q: %w",
				rewrite.Domain,
				err,
			)
		}

		result.Added++
	}

	for _, change := range plan.Update {
		if err := client.UpdateRewrite(
			ctx,
			change.Current,
			change.Desired,
		); err != nil {
			return result, fmt.Errorf(
				"execute rewrite update for %q: %w",
				change.Desired.Domain,
				err,
			)
		}

		result.Updated++
	}

	// Deletions are deliberately performed last.  This ensures that new and
	// updated records are in place before stale managed records are removed.
	for _, rewrite := range plan.Delete {
		if err := client.DeleteRewrite(
			ctx,
			rewrite,
		); err != nil {
			return result, fmt.Errorf(
				"execute rewrite deletion for %q: %w",
				rewrite.Domain,
				err,
			)
		}

		result.Deleted++
	}

	return result, nil
}

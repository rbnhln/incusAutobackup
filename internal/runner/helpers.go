package runner

import (
	"log/slog"

	"github.com/rbnhln/incusAutobackup/internal/retention"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

func planLog(logger *slog.Logger, plan retention.PrunePlan, kind transfer.Kind, subject, policy, role string, dryRun bool) error {
	if len(plan.Future) > 0 {
		logger.Warn("found IAB snapshots with timestamps in the future, keeping them", "role", role, "count", len(plan.Future))
	}

	if len(plan.Remove) > 0 {
		logger.Info("prune result",
			"role", role,
			"kind", kind,
			"subject", subject,
			"policy", policy,
			"dryRun", dryRun,
			"keep", len(plan.Keep),
			"remove", len(plan.Remove),
			"unmanaged", len(plan.Unmanaged),
		)
	} else {
		logger.Debug("prune result (nothing to remove)",
			"role", role,
			"kind", kind,
			"subject", subject,
			"policy", policy,
			"dryRun", dryRun,
			"keep", len(plan.Keep),
			"remove", 0,
			"unmanaged", len(plan.Unmanaged),
		)
	}
	return nil
}

package trialchecker

import (
	"context"
	"log/slog"
	"time"

	"github.com/usememos/memos/store"
)

type Runner struct {
	Store *store.Store
}

func NewRunner(store *store.Store) *Runner {
	return &Runner{
		Store: store,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	slog.Info("Starting trial period check")

	users, err := r.Store.ListUsers(ctx, &store.FindUser{})
	if err != nil {
		slog.Error("Failed to list users", "error", err)
		return err
	}

	now := time.Now().Unix()
	downgradedCount := 0

	for _, user := range users {
		vipStatus, err := r.Store.GetUserVIPStatus(ctx, user.ID)
		if err != nil {
			slog.Error("Failed to get VIP status", "userID", user.ID, "error", err)
			continue
		}

		if vipStatus == nil || vipStatus.VipType != store.VipTypeTrial {
			continue
		}

		if vipStatus.TrialEndTs != nil && *vipStatus.TrialEndTs < now {
			_, err := r.Store.UpdateUserVIPStatus(ctx, &store.UpdateUserVIPStatus{
				UserID:  user.ID,
				IsVIP:   ptrBool(false),
				VipType: ptrVipType(store.VipTypeNone),
			})
			if err != nil {
				slog.Error("Failed to downgrade VIP status", "userID", user.ID, "error", err)
				continue
			}

			_, err = r.Store.UpdateUserStorageUsage(ctx, &store.UpdateUserStorageUsage{
				UserID:     user.ID,
				QuotaBytes: ptrInt64(store.DefaultQuotaBytes),
			})
			if err != nil {
				slog.Error("Failed to update storage quota", "userID", user.ID, "error", err)
				continue
			}

			_, err = r.Store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
				UserID:    user.ID,
				EventType: "TRIAL_END",
				EventTs:   now,
			})
			if err != nil {
				slog.Error("Failed to record trial end", "userID", user.ID, "error", err)
			}

			downgradedCount++
			slog.Info("Downgraded user from trial to normal", "userID", user.ID)
		}
	}

	slog.Info("Trial period check completed", "downgradedCount", downgradedCount)
	return nil
}

func ptrBool(v bool) *bool {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrVipType(v store.VipType) *store.VipType {
	return &v
}

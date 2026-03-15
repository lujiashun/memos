package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/usememos/memos/store"
)

func (d *DB) CreateUserSubscription(ctx context.Context, create *store.UserSubscription) (*store.UserSubscription, error) {
	query := `
		INSERT INTO user_subscription (
			user_id,
			original_transaction_id,
			transaction_id,
			product_id,
			status,
			purchase_date_ts,
			expires_date_ts,
			cancellation_date_ts,
			grace_period_expires_ts,
			is_trial_period,
			is_in_intro_offer,
			environment,
			last_notification_type,
			last_notification_ts,
			receipt_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_ts, updated_ts
	`
	err := d.db.QueryRowContext(
		ctx,
		query,
		create.UserID,
		create.OriginalTransactionID,
		create.TransactionID,
		create.ProductID,
		create.Status,
		create.PurchaseDateTs,
		create.ExpiresDateTs,
		create.CancellationDateTs,
		create.GracePeriodExpiresTs,
		create.IsTrialPeriod,
		create.IsInIntroOffer,
		create.Environment,
		create.LastNotificationType,
		create.LastNotificationTs,
		create.ReceiptData,
	).Scan(&create.ID, &create.CreatedTs, &create.UpdatedTs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user subscription")
	}

	return create, nil
}

func (d *DB) GetUserSubscription(ctx context.Context, find *store.FindUserSubscription) (*store.UserSubscription, error) {
	list, err := d.ListUserSubscriptions(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func (d *DB) ListUserSubscriptions(ctx context.Context, find *store.FindUserSubscription) ([]*store.UserSubscription, error) {
	where, args := []string{"1 = 1"}, []any{}
	argID := 1
	if find.ID != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", argID)), append(args, *find.ID)
		argID++
	}
	if find.UserID != nil {
		where, args = append(where, fmt.Sprintf("user_id = $%d", argID)), append(args, *find.UserID)
		argID++
	}
	if find.OriginalTransactionID != nil {
		where, args = append(where, fmt.Sprintf("original_transaction_id = $%d", argID)), append(args, *find.OriginalTransactionID)
		argID++
	}
	if find.Status != nil {
		where, args = append(where, fmt.Sprintf("status = $%d", argID)), append(args, *find.Status)
		argID++
	}

	query := `
		SELECT 
			id,
			user_id,
			original_transaction_id,
			transaction_id,
			product_id,
			status,
			purchase_date_ts,
			expires_date_ts,
			cancellation_date_ts,
			grace_period_expires_ts,
			is_trial_period,
			is_in_intro_offer,
			environment,
			last_notification_type,
			last_notification_ts,
			receipt_data,
			created_ts,
			updated_ts
		FROM user_subscription
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_ts DESC
	`
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list user subscriptions")
	}
	defer rows.Close()

	var subscriptions []*store.UserSubscription
	for rows.Next() {
		subscription := &store.UserSubscription{}
		var cancellationDateTs, gracePeriodExpiresTs sql.NullInt64
		var lastNotificationType, receiptData sql.NullString
		var lastNotificationTs sql.NullInt64

		err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.OriginalTransactionID,
			&subscription.TransactionID,
			&subscription.ProductID,
			&subscription.Status,
			&subscription.PurchaseDateTs,
			&subscription.ExpiresDateTs,
			&cancellationDateTs,
			&gracePeriodExpiresTs,
			&subscription.IsTrialPeriod,
			&subscription.IsInIntroOffer,
			&subscription.Environment,
			&lastNotificationType,
			&lastNotificationTs,
			&receiptData,
			&subscription.CreatedTs,
			&subscription.UpdatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan user subscription")
		}

		if cancellationDateTs.Valid {
			subscription.CancellationDateTs = &cancellationDateTs.Int64
		}
		if gracePeriodExpiresTs.Valid {
			subscription.GracePeriodExpiresTs = &gracePeriodExpiresTs.Int64
		}
		if lastNotificationType.Valid {
			subscription.LastNotificationType = &lastNotificationType.String
		}
		if lastNotificationTs.Valid {
			subscription.LastNotificationTs = &lastNotificationTs.Int64
		}
		if receiptData.Valid {
			subscription.ReceiptData = &receiptData.String
		}

		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate user subscriptions")
	}

	return subscriptions, nil
}

func (d *DB) UpdateUserSubscription(ctx context.Context, update *store.UpdateUserSubscription) (*store.UserSubscription, error) {
	set, args := []string{"updated_ts = EXTRACT(EPOCH FROM NOW())"}, []any{}
	argID := 1
	if update.TransactionID != nil {
		set, args = append(set, fmt.Sprintf("transaction_id = $%d", argID)), append(args, *update.TransactionID)
		argID++
	}
	if update.Status != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", argID)), append(args, *update.Status)
		argID++
	}
	if update.ExpiresDateTs != nil {
		set, args = append(set, fmt.Sprintf("expires_date_ts = $%d", argID)), append(args, *update.ExpiresDateTs)
		argID++
	}
	if update.CancellationDateTs != nil {
		set, args = append(set, fmt.Sprintf("cancellation_date_ts = $%d", argID)), append(args, *update.CancellationDateTs)
		argID++
	}
	if update.GracePeriodExpiresTs != nil {
		set, args = append(set, fmt.Sprintf("grace_period_expires_ts = $%d", argID)), append(args, *update.GracePeriodExpiresTs)
		argID++
	}
	if update.LastNotificationType != nil {
		set, args = append(set, fmt.Sprintf("last_notification_type = $%d", argID)), append(args, *update.LastNotificationType)
		argID++
	}
	if update.LastNotificationTs != nil {
		set, args = append(set, fmt.Sprintf("last_notification_ts = $%d", argID)), append(args, *update.LastNotificationTs)
		argID++
	}
	if update.ReceiptData != nil {
		set, args = append(set, fmt.Sprintf("receipt_data = $%d", argID)), append(args, *update.ReceiptData)
		argID++
	}

	args = append(args, update.ID)
	query := `
		UPDATE user_subscription
		SET ` + strings.Join(set, ", ") + fmt.Sprintf(" WHERE id = $%d", argID) + `
		RETURNING 
			id, user_id, original_transaction_id, transaction_id, product_id, status,
			purchase_date_ts, expires_date_ts, cancellation_date_ts, grace_period_expires_ts,
			is_trial_period, is_in_intro_offer, environment, last_notification_type,
			last_notification_ts, receipt_data, created_ts, updated_ts
	`
	subscription := &store.UserSubscription{}
	var cancellationDateTs, gracePeriodExpiresTs sql.NullInt64
	var lastNotificationType, receiptData sql.NullString
	var lastNotificationTs sql.NullInt64

	err := d.db.QueryRowContext(ctx, query, args...).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.OriginalTransactionID,
		&subscription.TransactionID,
		&subscription.ProductID,
		&subscription.Status,
		&subscription.PurchaseDateTs,
		&subscription.ExpiresDateTs,
		&cancellationDateTs,
		&gracePeriodExpiresTs,
		&subscription.IsTrialPeriod,
		&subscription.IsInIntroOffer,
		&subscription.Environment,
		&lastNotificationType,
		&lastNotificationTs,
		&receiptData,
		&subscription.CreatedTs,
		&subscription.UpdatedTs,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user subscription")
	}

	if cancellationDateTs.Valid {
		subscription.CancellationDateTs = &cancellationDateTs.Int64
	}
	if gracePeriodExpiresTs.Valid {
		subscription.GracePeriodExpiresTs = &gracePeriodExpiresTs.Int64
	}
	if lastNotificationType.Valid {
		subscription.LastNotificationType = &lastNotificationType.String
	}
	if lastNotificationTs.Valid {
		subscription.LastNotificationTs = &lastNotificationTs.Int64
	}
	if receiptData.Valid {
		subscription.ReceiptData = &receiptData.String
	}

	return subscription, nil
}

func (d *DB) DeleteUserSubscription(ctx context.Context, delete *store.DeleteUserSubscription) error {
	query := `DELETE FROM user_subscription WHERE id = $1`
	result, err := d.db.ExecContext(ctx, query, delete.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete user subscription")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return errors.New("user subscription not found")
	}
	return nil
}

func (d *DB) CreateUserVIPStatus(ctx context.Context, create *store.UserVIPStatus) (*store.UserVIPStatus, error) {
	query := `
		INSERT INTO user_vip_status (
			user_id,
			is_vip,
			vip_type,
			trial_start_ts,
			trial_end_ts,
			trial_used,
			subscription_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_ts, updated_ts
	`
	err := d.db.QueryRowContext(
		ctx,
		query,
		create.UserID,
		create.IsVIP,
		create.VipType,
		create.TrialStartTs,
		create.TrialEndTs,
		create.TrialUsed,
		create.SubscriptionID,
	).Scan(&create.ID, &create.CreatedTs, &create.UpdatedTs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user VIP status")
	}

	return create, nil
}

func (d *DB) GetUserVIPStatus(ctx context.Context, userID int32) (*store.UserVIPStatus, error) {
	query := `
		SELECT 
			id,
			user_id,
			is_vip,
			vip_type,
			trial_start_ts,
			trial_end_ts,
			trial_used,
			subscription_id,
			created_ts,
			updated_ts
		FROM user_vip_status
		WHERE user_id = $1
	`
	vipStatus := &store.UserVIPStatus{}
	var trialStartTs, trialEndTs sql.NullInt64
	var subscriptionID sql.NullInt32

	err := d.db.QueryRowContext(ctx, query, userID).Scan(
		&vipStatus.ID,
		&vipStatus.UserID,
		&vipStatus.IsVIP,
		&vipStatus.VipType,
		&trialStartTs,
		&trialEndTs,
		&vipStatus.TrialUsed,
		&subscriptionID,
		&vipStatus.CreatedTs,
		&vipStatus.UpdatedTs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user VIP status")
	}

	if trialStartTs.Valid {
		vipStatus.TrialStartTs = &trialStartTs.Int64
	}
	if trialEndTs.Valid {
		vipStatus.TrialEndTs = &trialEndTs.Int64
	}
	if subscriptionID.Valid {
		vipStatus.SubscriptionID = &subscriptionID.Int32
	}

	return vipStatus, nil
}

func (d *DB) UpdateUserVIPStatus(ctx context.Context, update *store.UpdateUserVIPStatus) (*store.UserVIPStatus, error) {
	set, args := []string{"updated_ts = EXTRACT(EPOCH FROM NOW())"}, []any{}
	argID := 1
	if update.IsVIP != nil {
		set, args = append(set, fmt.Sprintf("is_vip = $%d", argID)), append(args, *update.IsVIP)
		argID++
	}
	if update.VipType != nil {
		set, args = append(set, fmt.Sprintf("vip_type = $%d", argID)), append(args, *update.VipType)
		argID++
	}
	if update.TrialStartTs != nil {
		set, args = append(set, fmt.Sprintf("trial_start_ts = $%d", argID)), append(args, *update.TrialStartTs)
		argID++
	}
	if update.TrialEndTs != nil {
		set, args = append(set, fmt.Sprintf("trial_end_ts = $%d", argID)), append(args, *update.TrialEndTs)
		argID++
	}
	if update.TrialUsed != nil {
		set, args = append(set, fmt.Sprintf("trial_used = $%d", argID)), append(args, *update.TrialUsed)
		argID++
	}
	if update.SubscriptionID != nil {
		set, args = append(set, fmt.Sprintf("subscription_id = $%d", argID)), append(args, *update.SubscriptionID)
		argID++
	}

	args = append(args, update.UserID)
	query := `
		UPDATE user_vip_status
		SET ` + strings.Join(set, ", ") + fmt.Sprintf(" WHERE user_id = $%d", argID) + `
		RETURNING 
			id, user_id, is_vip, vip_type, trial_start_ts, trial_end_ts,
			trial_used, subscription_id, created_ts, updated_ts
	`
	vipStatus := &store.UserVIPStatus{}
	var trialStartTs, trialEndTs sql.NullInt64
	var subscriptionID sql.NullInt32

	err := d.db.QueryRowContext(ctx, query, args...).Scan(
		&vipStatus.ID,
		&vipStatus.UserID,
		&vipStatus.IsVIP,
		&vipStatus.VipType,
		&trialStartTs,
		&trialEndTs,
		&vipStatus.TrialUsed,
		&subscriptionID,
		&vipStatus.CreatedTs,
		&vipStatus.UpdatedTs,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user VIP status")
	}

	if trialStartTs.Valid {
		vipStatus.TrialStartTs = &trialStartTs.Int64
	}
	if trialEndTs.Valid {
		vipStatus.TrialEndTs = &trialEndTs.Int64
	}
	if subscriptionID.Valid {
		vipStatus.SubscriptionID = &subscriptionID.Int32
	}

	return vipStatus, nil
}

func (d *DB) CreateUserStorageUsage(ctx context.Context, create *store.UserStorageUsage) (*store.UserStorageUsage, error) {
	query := `
		INSERT INTO user_storage_usage (
			user_id,
			total_bytes,
			attachment_bytes,
			memo_content_bytes,
			quota_bytes,
			last_calculated_ts
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_ts, updated_ts
	`
	err := d.db.QueryRowContext(
		ctx,
		query,
		create.UserID,
		create.TotalBytes,
		create.AttachmentBytes,
		create.MemoContentBytes,
		create.QuotaBytes,
		create.LastCalculatedTs,
	).Scan(&create.ID, &create.CreatedTs, &create.UpdatedTs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user storage usage")
	}

	return create, nil
}

func (d *DB) GetUserStorageUsage(ctx context.Context, userID int32) (*store.UserStorageUsage, error) {
	query := `
		SELECT 
			id,
			user_id,
			total_bytes,
			attachment_bytes,
			memo_content_bytes,
			quota_bytes,
			last_calculated_ts,
			created_ts,
			updated_ts
		FROM user_storage_usage
		WHERE user_id = $1
	`
	usage := &store.UserStorageUsage{}
	err := d.db.QueryRowContext(ctx, query, userID).Scan(
		&usage.ID,
		&usage.UserID,
		&usage.TotalBytes,
		&usage.AttachmentBytes,
		&usage.MemoContentBytes,
		&usage.QuotaBytes,
		&usage.LastCalculatedTs,
		&usage.CreatedTs,
		&usage.UpdatedTs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user storage usage")
	}

	return usage, nil
}

func (d *DB) UpdateUserStorageUsage(ctx context.Context, update *store.UpdateUserStorageUsage) (*store.UserStorageUsage, error) {
	set, args := []string{"updated_ts = EXTRACT(EPOCH FROM NOW())"}, []any{}
	argID := 1
	if update.TotalBytes != nil {
		set, args = append(set, fmt.Sprintf("total_bytes = $%d", argID)), append(args, *update.TotalBytes)
		argID++
	}
	if update.AttachmentBytes != nil {
		set, args = append(set, fmt.Sprintf("attachment_bytes = $%d", argID)), append(args, *update.AttachmentBytes)
		argID++
	}
	if update.MemoContentBytes != nil {
		set, args = append(set, fmt.Sprintf("memo_content_bytes = $%d", argID)), append(args, *update.MemoContentBytes)
		argID++
	}
	if update.QuotaBytes != nil {
		set, args = append(set, fmt.Sprintf("quota_bytes = $%d", argID)), append(args, *update.QuotaBytes)
		argID++
	}
	if update.LastCalculatedTs != nil {
		set, args = append(set, fmt.Sprintf("last_calculated_ts = $%d", argID)), append(args, *update.LastCalculatedTs)
		argID++
	}

	args = append(args, update.UserID)
	query := `
		UPDATE user_storage_usage
		SET ` + strings.Join(set, ", ") + fmt.Sprintf(" WHERE user_id = $%d", argID) + `
		RETURNING 
			id, user_id, total_bytes, attachment_bytes, memo_content_bytes,
			quota_bytes, last_calculated_ts, created_ts, updated_ts
	`
	usage := &store.UserStorageUsage{}
	err := d.db.QueryRowContext(ctx, query, args...).Scan(
		&usage.ID,
		&usage.UserID,
		&usage.TotalBytes,
		&usage.AttachmentBytes,
		&usage.MemoContentBytes,
		&usage.QuotaBytes,
		&usage.LastCalculatedTs,
		&usage.CreatedTs,
		&usage.UpdatedTs,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user storage usage")
	}

	return usage, nil
}

func (d *DB) CreateSubscriptionHistory(ctx context.Context, create *store.SubscriptionHistory) (*store.SubscriptionHistory, error) {
	query := `
		INSERT INTO subscription_history (
			user_id,
			event_type,
			event_ts,
			original_transaction_id,
			product_id,
			from_status,
			to_status,
			details
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_ts
	`
	err := d.db.QueryRowContext(
		ctx,
		query,
		create.UserID,
		create.EventType,
		create.EventTs,
		create.OriginalTransactionID,
		create.ProductID,
		create.FromStatus,
		create.ToStatus,
		create.Details,
	).Scan(&create.ID, &create.CreatedTs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create subscription history")
	}

	return create, nil
}

func (d *DB) ListSubscriptionHistory(ctx context.Context, find *store.FindSubscriptionHistory) ([]*store.SubscriptionHistory, error) {
	where, args := []string{"1 = 1"}, []any{}
	argID := 1
	if find.UserID != nil {
		where, args = append(where, fmt.Sprintf("user_id = $%d", argID)), append(args, *find.UserID)
		argID++
	}
	if find.EventType != nil {
		where, args = append(where, fmt.Sprintf("event_type = $%d", argID)), append(args, *find.EventType)
		argID++
	}

	limit := 100
	if find.Limit != nil {
		limit = *find.Limit
	}

	offset := 0
	if find.Offset != nil {
		offset = *find.Offset
	}

	query := fmt.Sprintf(`
		SELECT 
			id,
			user_id,
			event_type,
			event_ts,
			original_transaction_id,
			product_id,
			from_status,
			to_status,
			details,
			created_ts
		FROM subscription_history
		WHERE %s
		ORDER BY event_ts DESC
		LIMIT %d OFFSET %d
	`, strings.Join(where, " AND "), limit, offset)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list subscription history")
	}
	defer rows.Close()

	var history []*store.SubscriptionHistory
	for rows.Next() {
		event := &store.SubscriptionHistory{}
		var originalTransactionID, productID, fromStatus, toStatus, details sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.EventType,
			&event.EventTs,
			&originalTransactionID,
			&productID,
			&fromStatus,
			&toStatus,
			&details,
			&event.CreatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan subscription history")
		}

		if originalTransactionID.Valid {
			event.OriginalTransactionID = &originalTransactionID.String
		}
		if productID.Valid {
			event.ProductID = &productID.String
		}
		if fromStatus.Valid {
			event.FromStatus = &fromStatus.String
		}
		if toStatus.Valid {
			event.ToStatus = &toStatus.String
		}
		if details.Valid {
			event.Details = &details.String
		}

		history = append(history, event)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate subscription history")
	}

	return history, nil
}

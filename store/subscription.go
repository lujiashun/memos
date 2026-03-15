package store

import (
	"context"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive       SubscriptionStatus = "ACTIVE"
	SubscriptionStatusExpired      SubscriptionStatus = "EXPIRED"
	SubscriptionStatusCancelled    SubscriptionStatus = "CANCELLED"
	SubscriptionStatusGracePeriod  SubscriptionStatus = "GRACE_PERIOD"
	SubscriptionStatusBillingRetry SubscriptionStatus = "BILLING_RETRY"
)

type VipType string

const (
	VipTypeNone         VipType = "NONE"
	VipTypeTrial        VipType = "TRIAL"
	VipTypeSubscription VipType = "SUBSCRIPTION"
)

type UserSubscription struct {
	ID int32

	UserID int32

	OriginalTransactionID string
	TransactionID         string
	ProductID             string

	Status SubscriptionStatus

	PurchaseDateTs       int64
	ExpiresDateTs        int64
	CancellationDateTs   *int64
	GracePeriodExpiresTs *int64

	IsTrialPeriod  bool
	IsInIntroOffer bool

	Environment string

	LastNotificationType *string
	LastNotificationTs   *int64

	ReceiptData *string

	CreatedTs int64
	UpdatedTs int64
}

type UserVIPStatus struct {
	ID int32

	UserID int32

	IsVIP   bool
	VipType VipType

	TrialStartTs *int64
	TrialEndTs   *int64
	TrialUsed    bool

	SubscriptionID *int32

	CreatedTs int64
	UpdatedTs int64
}

type UserStorageUsage struct {
	ID int32

	UserID int32

	TotalBytes       int64
	AttachmentBytes  int64
	MemoContentBytes int64

	QuotaBytes int64

	LastCalculatedTs int64

	CreatedTs int64
	UpdatedTs int64
}

type SubscriptionHistory struct {
	ID int32

	UserID int32

	EventType string
	EventTs   int64

	OriginalTransactionID *string
	ProductID             *string

	FromStatus *string
	ToStatus   *string

	Details *string

	CreatedTs int64
}

type UpdateUserSubscription struct {
	ID int32

	TransactionID        *string
	Status               *SubscriptionStatus
	ExpiresDateTs        *int64
	CancellationDateTs   *int64
	GracePeriodExpiresTs *int64
	LastNotificationType *string
	LastNotificationTs   *int64
	ReceiptData          *string
}

type FindUserSubscription struct {
	ID                    *int32
	UserID                *int32
	OriginalTransactionID *string
	Status                *SubscriptionStatus
}

type DeleteUserSubscription struct {
	ID int32
}

type UpdateUserVIPStatus struct {
	UserID int32

	IsVIP          *bool
	VipType        *VipType
	TrialStartTs   *int64
	TrialEndTs     *int64
	TrialUsed      *bool
	SubscriptionID *int32
}

type UpdateUserStorageUsage struct {
	UserID int32

	TotalBytes       *int64
	AttachmentBytes  *int64
	MemoContentBytes *int64
	QuotaBytes       *int64
	LastCalculatedTs *int64
}

type FindSubscriptionHistory struct {
	UserID    *int32
	EventType *string
	Limit     *int
	Offset    *int
}

const (
	DefaultQuotaBytes  = 50 * 1024 * 1024
	VIPQuotaBytes      = 5 * 1024 * 1024 * 1024
	TrialDurationDays  = 10
)

func (s *Store) CreateUserSubscription(ctx context.Context, create *UserSubscription) (*UserSubscription, error) {
	return s.driver.CreateUserSubscription(ctx, create)
}

func (s *Store) GetUserSubscription(ctx context.Context, find *FindUserSubscription) (*UserSubscription, error) {
	return s.driver.GetUserSubscription(ctx, find)
}

func (s *Store) ListUserSubscriptions(ctx context.Context, find *FindUserSubscription) ([]*UserSubscription, error) {
	return s.driver.ListUserSubscriptions(ctx, find)
}

func (s *Store) UpdateUserSubscription(ctx context.Context, update *UpdateUserSubscription) (*UserSubscription, error) {
	return s.driver.UpdateUserSubscription(ctx, update)
}

func (s *Store) DeleteUserSubscription(ctx context.Context, delete *DeleteUserSubscription) error {
	return s.driver.DeleteUserSubscription(ctx, delete)
}

func (s *Store) CreateUserVIPStatus(ctx context.Context, create *UserVIPStatus) (*UserVIPStatus, error) {
	return s.driver.CreateUserVIPStatus(ctx, create)
}

func (s *Store) GetUserVIPStatus(ctx context.Context, userID int32) (*UserVIPStatus, error) {
	return s.driver.GetUserVIPStatus(ctx, userID)
}

func (s *Store) UpdateUserVIPStatus(ctx context.Context, update *UpdateUserVIPStatus) (*UserVIPStatus, error) {
	return s.driver.UpdateUserVIPStatus(ctx, update)
}

func (s *Store) CreateUserStorageUsage(ctx context.Context, create *UserStorageUsage) (*UserStorageUsage, error) {
	return s.driver.CreateUserStorageUsage(ctx, create)
}

func (s *Store) GetUserStorageUsage(ctx context.Context, userID int32) (*UserStorageUsage, error) {
	return s.driver.GetUserStorageUsage(ctx, userID)
}

func (s *Store) UpdateUserStorageUsage(ctx context.Context, update *UpdateUserStorageUsage) (*UserStorageUsage, error) {
	return s.driver.UpdateUserStorageUsage(ctx, update)
}

func (s *Store) CreateSubscriptionHistory(ctx context.Context, create *SubscriptionHistory) (*SubscriptionHistory, error) {
	return s.driver.CreateSubscriptionHistory(ctx, create)
}

func (s *Store) ListSubscriptionHistory(ctx context.Context, find *FindSubscriptionHistory) ([]*SubscriptionHistory, error) {
	return s.driver.ListSubscriptionHistory(ctx, find)
}

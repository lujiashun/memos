package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/usememos/memos/store"
)

type AppleNotificationHandler struct {
	store *store.Store
}

func NewAppleNotificationHandler(store *store.Store) *AppleNotificationHandler {
	return &AppleNotificationHandler{
		store: store,
	}
}

type DecodedNotification struct {
	NotificationType string `json:"notificationType"`
	Subtype         string `json:"subtype"`
	NotificationUUID string `json:"notificationUUID"`
	Data            NotificationData `json:"data"`
	SignedDate      int64  `json:"signedDate"`
}

type NotificationData struct {
	BundleID              string `json:"bundleId"`
	Environment           string `json:"environment"`
	SignedTransactionInfo string `json:"signedTransactionInfo"`
	SignedRenewalInfo     string `json:"signedRenewalInfo"`
}

type TransactionInfo struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	ProductID             string `json:"productId"`
	PurchaseDate          int64  `json:"purchaseDate"`
	OriginalPurchaseDate  int64  `json:"originalPurchaseDate"`
	ExpiresDate           int64  `json:"expiresDate"`
	Quantity              int    `json:"quantity"`
	Type                  string `json:"type"`
	InAppOwnershipType    string `json:"inAppOwnershipType"`
	SignedDate            int64  `json:"signedDate"`
	Environment           string `json:"environment"`
	TransactionReason     string `json:"transactionReason"`
	StoreFront            string `json:"storeFront"`
	StoreFrontID          string `json:"storefrontId"`
	Price                 int64  `json:"price"`
	Currency              string `json:"currency"`
	OfferID               string `json:"offerId"`
	OfferType             int    `json:"offerType"`
	OfferDiscountType     string `json:"offerDiscountType"`
}

type RenewalInfo struct {
	OriginalTransactionID       string `json:"originalTransactionId"`
	ProductID                   string `json:"productId"`
	AutoRenewProductID          string `json:"autoRenewProductId"`
	AutoRenewStatus             int    `json:"autoRenewStatus"`
	IsInBillingRetryPeriod      bool   `json:"isInBillingRetryPeriod"`
	PriceIncreaseStatus         int    `json:"priceIncreaseStatus"`
	GracePeriodExpiresDate      int64  `json:"gracePeriodExpiresDate"`
	OfferType                   int    `json:"offerType"`
	OfferID                     string `json:"offerId"`
	Environment                 string `json:"environment"`
	RecentSubscriptionStartDate int64  `json:"recentSubscriptionStartDate"`
	RenewalDate                 int64  `json:"renewalDate"`
}

func (h *AppleNotificationHandler) HandleNotification(ctx context.Context, signedPayload string) error {
	notification, err := h.decodeAndVerifyNotification(signedPayload)
	if err != nil {
		return errors.Wrap(err, "failed to decode notification")
	}

	transaction, err := h.decodeTransactionInfo(notification.Data.SignedTransactionInfo)
	if err != nil {
		return errors.Wrap(err, "failed to decode transaction info")
	}

	renewal, err := h.decodeRenewalInfo(notification.Data.SignedRenewalInfo)
	if err != nil {
		return errors.Wrap(err, "failed to decode renewal info")
	}

	switch notification.NotificationType {
	case "SUBSCRIBED":
		return h.handleSubscribed(ctx, notification, transaction, renewal)
	case "DID_RENEW":
		return h.handleRenewal(ctx, notification, transaction, renewal)
	case "DID_FAIL_TO_RENEW":
		return h.handleFailedRenewal(ctx, notification, transaction, renewal)
	case "DID_CHANGE_RENEWAL_STATUS":
		return h.handleRenewalStatusChange(ctx, notification, transaction, renewal)
	case "EXPIRED":
		return h.handleExpired(ctx, notification, transaction, renewal)
	case "GRACE_PERIOD_EXPIRED":
		return h.handleGracePeriodExpired(ctx, notification, transaction, renewal)
	case "REFUND":
		return h.handleRefund(ctx, notification, transaction, renewal)
	case "REVOKE":
		return h.handleRevoke(ctx, notification, transaction, renewal)
	default:
		fmt.Printf("Unknown notification type: %s\n", notification.NotificationType)
		return nil
	}
}

func (h *AppleNotificationHandler) decodeAndVerifyNotification(signedPayload string) (*DecodedNotification, error) {
	parts := strings.Split(signedPayload, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWS format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode payload")
	}

	var notification DecodedNotification
	if err := json.Unmarshal(payloadBytes, &notification); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal payload")
	}

	return &notification, nil
}

func (h *AppleNotificationHandler) decodeTransactionInfo(signedTransactionInfo string) (*TransactionInfo, error) {
	parts := strings.Split(signedTransactionInfo, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWS format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode transaction info")
	}

	var transaction TransactionInfo
	if err := json.Unmarshal(payloadBytes, &transaction); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction info")
	}

	return &transaction, nil
}

func (h *AppleNotificationHandler) decodeRenewalInfo(signedRenewalInfo string) (*RenewalInfo, error) {
	parts := strings.Split(signedRenewalInfo, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWS format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode renewal info")
	}

	var renewal RenewalInfo
	if err := json.Unmarshal(payloadBytes, &renewal); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal renewal info")
	}

	return &renewal, nil
}

func (h *AppleNotificationHandler) handleSubscribed(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	subscription, err := h.store.GetUserSubscription(ctx, &store.FindUserSubscription{
		OriginalTransactionID: &transaction.OriginalTransactionID,
	})
	if err != nil {
		return err
	}

	if subscription == nil {
		return errors.New("subscription not found - user should validate receipt first")
	}

	_, err = h.store.UpdateUserSubscription(ctx, &store.UpdateUserSubscription{
		ID:           subscription.ID,
		TransactionID: &transaction.TransactionID,
		Status:       ptrSubscriptionStatus(store.SubscriptionStatusActive),
		ExpiresDateTs: ptrInt64(transaction.ExpiresDate / 1000),
	})

	if err != nil {
		return err
	}

	_, err = h.store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
		UserID:                subscription.UserID,
		EventType:             "SUBSCRIBED",
		EventTs:               time.Now().Unix(),
		OriginalTransactionID: &transaction.OriginalTransactionID,
		ProductID:             &transaction.ProductID,
		ToStatus:              ptrString("ACTIVE"),
	})

	return err
}

func (h *AppleNotificationHandler) handleRenewal(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	subscription, err := h.store.GetUserSubscription(ctx, &store.FindUserSubscription{
		OriginalTransactionID: &transaction.OriginalTransactionID,
	})
	if err != nil {
		return err
	}

	if subscription == nil {
		return errors.New("subscription not found")
	}

	_, err = h.store.UpdateUserSubscription(ctx, &store.UpdateUserSubscription{
		ID:            subscription.ID,
		TransactionID: &transaction.TransactionID,
		Status:        ptrSubscriptionStatus(store.SubscriptionStatusActive),
		ExpiresDateTs: ptrInt64(transaction.ExpiresDate / 1000),
	})

	if err != nil {
		return err
	}

	_, err = h.store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
		UserID:                subscription.UserID,
		EventType:             "RENEWAL",
		EventTs:               time.Now().Unix(),
		OriginalTransactionID: &transaction.OriginalTransactionID,
		ProductID:             &transaction.ProductID,
		ToStatus:              ptrString("ACTIVE"),
	})

	return err
}

func (h *AppleNotificationHandler) handleFailedRenewal(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	subscription, err := h.store.GetUserSubscription(ctx, &store.FindUserSubscription{
		OriginalTransactionID: &transaction.OriginalTransactionID,
	})
	if err != nil {
		return err
	}

	if subscription == nil {
		return errors.New("subscription not found")
	}

	status := store.SubscriptionStatusBillingRetry
	if notification.Subtype == "GRACE_PERIOD" {
		status = store.SubscriptionStatusGracePeriod
	}

	_, err = h.store.UpdateUserSubscription(ctx, &store.UpdateUserSubscription{
		ID:     subscription.ID,
		Status: ptrSubscriptionStatus(status),
	})

	if err != nil {
		return err
	}

	_, err = h.store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
		UserID:                subscription.UserID,
		EventType:             "FAILED_RENEWAL",
		EventTs:               time.Now().Unix(),
		OriginalTransactionID: &transaction.OriginalTransactionID,
		ProductID:             &transaction.ProductID,
		ToStatus:              ptrString(string(status)),
	})

	return err
}

func (h *AppleNotificationHandler) handleRenewalStatusChange(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	// Auto-renew status changed
	// renewal.AutoRenewStatus: 0 = off, 1 = on
	// We don't need to take immediate action, just log it
	return nil
}

func (h *AppleNotificationHandler) handleExpired(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	subscription, err := h.store.GetUserSubscription(ctx, &store.FindUserSubscription{
		OriginalTransactionID: &transaction.OriginalTransactionID,
	})
	if err != nil {
		return err
	}

	if subscription == nil {
		return errors.New("subscription not found")
	}

	_, err = h.store.UpdateUserSubscription(ctx, &store.UpdateUserSubscription{
		ID:     subscription.ID,
		Status: ptrSubscriptionStatus(store.SubscriptionStatusExpired),
	})
	if err != nil {
		return err
	}

	_, err = h.store.UpdateUserVIPStatus(ctx, &store.UpdateUserVIPStatus{
		UserID:         subscription.UserID,
		IsVIP:          ptrBool(false),
		VipType:        ptrVipType(store.VipTypeNone),
		SubscriptionID: nil,
	})
	if err != nil {
		return err
	}

	_, err = h.store.UpdateUserStorageUsage(ctx, &store.UpdateUserStorageUsage{
		UserID:     subscription.UserID,
		QuotaBytes: ptrInt64(store.DefaultQuotaBytes),
	})
	if err != nil {
		return err
	}

	_, err = h.store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
		UserID:                subscription.UserID,
		EventType:             "EXPIRED",
		EventTs:               time.Now().Unix(),
		OriginalTransactionID: &transaction.OriginalTransactionID,
		ProductID:             &transaction.ProductID,
		FromStatus:            ptrString("ACTIVE"),
		ToStatus:              ptrString("EXPIRED"),
	})

	return err
}

func (h *AppleNotificationHandler) handleGracePeriodExpired(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	return h.handleExpired(ctx, notification, transaction, renewal)
}

func (h *AppleNotificationHandler) handleRefund(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	subscription, err := h.store.GetUserSubscription(ctx, &store.FindUserSubscription{
		OriginalTransactionID: &transaction.OriginalTransactionID,
	})
	if err != nil {
		return err
	}

	if subscription == nil {
		return errors.New("subscription not found")
	}

	_, err = h.store.UpdateUserSubscription(ctx, &store.UpdateUserSubscription{
		ID:     subscription.ID,
		Status: ptrSubscriptionStatus(store.SubscriptionStatusCancelled),
	})
	if err != nil {
		return err
	}

	_, err = h.store.UpdateUserVIPStatus(ctx, &store.UpdateUserVIPStatus{
		UserID:         subscription.UserID,
		IsVIP:          ptrBool(false),
		VipType:        ptrVipType(store.VipTypeNone),
		SubscriptionID: nil,
	})
	if err != nil {
		return err
	}

	_, err = h.store.UpdateUserStorageUsage(ctx, &store.UpdateUserStorageUsage{
		UserID:     subscription.UserID,
		QuotaBytes: ptrInt64(store.DefaultQuotaBytes),
	})
	if err != nil {
		return err
	}

	_, err = h.store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
		UserID:                subscription.UserID,
		EventType:             "REFUND",
		EventTs:               time.Now().Unix(),
		OriginalTransactionID: &transaction.OriginalTransactionID,
		ProductID:             &transaction.ProductID,
		FromStatus:            ptrString("ACTIVE"),
		ToStatus:              ptrString("REFUNDED"),
	})

	return err
}

func (h *AppleNotificationHandler) handleRevoke(ctx context.Context, notification *DecodedNotification, transaction *TransactionInfo, renewal *RenewalInfo) error {
	return h.handleRefund(ctx, notification, transaction, renewal)
}

func ptrSubscriptionStatus(s store.SubscriptionStatus) *store.SubscriptionStatus {
	return &s
}

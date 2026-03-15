package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

type SubscriptionService struct {
	v1pb.UnimplementedSubscriptionServiceServer
	store *store.Store
}

func NewSubscriptionService(store *store.Store) *SubscriptionService {
	return &SubscriptionService{
		store: store,
	}
}

// extractOrGetCurrentUserID extracts the user ID from the name, or returns the current authenticated user ID if name is "users/me"
func extractOrGetCurrentUserID(ctx context.Context, name string) (int32, error) {
	currentUserID, ok := ctx.Value(auth.UserIDContextKey).(int32)
	if !ok {
		return 0, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	if strings.HasSuffix(name, "/me") || name == "me" {
		return currentUserID, nil
	}

	userID, err := ExtractUserIDFromName(name)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid user name: %v", err)
	}

	if currentUserID != userID {
		return 0, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	return userID, nil
}

func (s *SubscriptionService) GetSubscriptionStatus(ctx context.Context, req *v1pb.GetSubscriptionStatusRequest) (*v1pb.SubscriptionStatus, error) {
	userID, err := extractOrGetCurrentUserID(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	vipStatus, err := s.store.GetUserVIPStatus(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get VIP status: %v", err)
	}

	storageUsage, err := s.store.GetUserStorageUsage(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get storage usage: %v", err)
	}

	response := &v1pb.SubscriptionStatus{
		Name:              fmt.Sprintf("users/%d", userID),
		IsVip:             vipStatus.IsVIP,
		VipType:           convertVipTypeToProto(vipStatus.VipType),
		StorageQuotaBytes: storageUsage.QuotaBytes,
		StorageUsedBytes:  storageUsage.TotalBytes,
		StorageExceeded:   storageUsage.TotalBytes > storageUsage.QuotaBytes,
	}

	if vipStatus.SubscriptionID != nil {
		subscription, err := s.store.GetUserSubscription(ctx, &store.FindUserSubscription{ID: vipStatus.SubscriptionID})
		if err == nil && subscription != nil {
			response.Subscription = &v1pb.Subscription{
				ProductId:             subscription.ProductID,
				State:                 convertSubscriptionStatusToProto(subscription.Status),
				PurchaseDate:          convertTimestamp(subscription.PurchaseDateTs),
				ExpiresDate:           convertTimestamp(subscription.ExpiresDateTs),
				IsTrial:               subscription.IsTrialPeriod,
				WillRenew:             subscription.Status == store.SubscriptionStatusActive,
				OriginalTransactionId: subscription.OriginalTransactionID,
			}
		}
	}

	response.TrialInfo = &v1pb.TrialInfo{
		TrialUsed: vipStatus.TrialUsed,
	}

	if vipStatus.TrialStartTs != nil && vipStatus.TrialEndTs != nil {
		response.TrialInfo.TrialStartDate = convertTimestamp(*vipStatus.TrialStartTs)
		response.TrialInfo.TrialEndDate = convertTimestamp(*vipStatus.TrialEndTs)

		endTime := time.Unix(*vipStatus.TrialEndTs, 0)
		remaining := time.Until(endTime)
		if remaining > 0 {
			response.TrialInfo.DaysRemaining = int32(remaining.Hours() / 24)
		}
	}

	return response, nil
}

func (s *SubscriptionService) ValidateReceipt(ctx context.Context, req *v1pb.ValidateReceiptRequest) (*v1pb.ValidateReceiptResponse, error) {
	userID, err := extractOrGetCurrentUserID(ctx, req.Parent)
	if err != nil {
		return nil, err
	}

	receiptResponse, err := s.validateReceiptWithApple(ctx, req.ReceiptData, req.Sandbox)
	if err != nil {
		return &v1pb.ValidateReceiptResponse{
			Status:       nil,
			Valid:        false,
			ErrorMessage: err.Error(),
		}, nil
	}

	_, err = s.processValidatedReceipt(ctx, userID, receiptResponse)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process receipt: %v", err)
	}

	statusReq := &v1pb.GetSubscriptionStatusRequest{
		Name: fmt.Sprintf("users/%d", userID),
	}
	statusResp, err := s.GetSubscriptionStatus(ctx, statusReq)
	if err != nil {
		return nil, err
	}

	return &v1pb.ValidateReceiptResponse{
		Status:       statusResp,
		Valid:        true,
		ErrorMessage: "",
	}, nil
}

func (s *SubscriptionService) RestorePurchases(ctx context.Context, req *v1pb.RestorePurchasesRequest) (*v1pb.RestorePurchasesResponse, error) {
	userID, err := extractOrGetCurrentUserID(ctx, req.Parent)
	if err != nil {
		return nil, err
	}

	subscriptions, err := s.store.ListUserSubscriptions(ctx, &store.FindUserSubscription{UserID: &userID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list subscriptions: %v", err)
	}

	var activeSubscription *store.UserSubscription
	for _, sub := range subscriptions {
		if sub.Status == store.SubscriptionStatusActive && sub.ExpiresDateTs > time.Now().Unix() {
			activeSubscription = sub
			break
		}
	}

	if activeSubscription != nil {
		_, err = s.store.UpdateUserVIPStatus(ctx, &store.UpdateUserVIPStatus{
			UserID:         userID,
			IsVIP:          ptrBool(true),
			VipType:        ptrVipType(store.VipTypeSubscription),
			SubscriptionID: ptrInt32(activeSubscription.ID),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update VIP status: %v", err)
		}

		_, err = s.store.UpdateUserStorageUsage(ctx, &store.UpdateUserStorageUsage{
			UserID:     userID,
			QuotaBytes: ptrInt64(store.VIPQuotaBytes),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update storage quota: %v", err)
		}
	}

	statusReq := &v1pb.GetSubscriptionStatusRequest{
		Name: fmt.Sprintf("users/%d", userID),
	}
	statusResp, err := s.GetSubscriptionStatus(ctx, statusReq)
	if err != nil {
		return nil, err
	}

	return &v1pb.RestorePurchasesResponse{
		Status:        statusResp,
		Restored:      activeSubscription != nil,
		RestoredCount: int32(len(subscriptions)),
	}, nil
}

func (s *SubscriptionService) GetStorageUsage(ctx context.Context, req *v1pb.GetStorageUsageRequest) (*v1pb.StorageUsage, error) {
	userID, err := extractOrGetCurrentUserID(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	usage, err := s.store.GetUserStorageUsage(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get storage usage: %v", err)
	}

	percentage := int32(0)
	if usage.QuotaBytes > 0 {
		percentage = int32((usage.TotalBytes * 100) / usage.QuotaBytes)
	}

	return &v1pb.StorageUsage{
		Name:           fmt.Sprintf("users/%d", userID),
		UsedBytes:      usage.TotalBytes,
		QuotaBytes:     usage.QuotaBytes,
		UsedPercentage: percentage,
		Breakdown: &v1pb.StorageBreakdown{
			AttachmentBytes:  usage.AttachmentBytes,
			MemoContentBytes: usage.MemoContentBytes,
		},
		QuotaExceeded: usage.TotalBytes > usage.QuotaBytes,
	}, nil
}

func (s *SubscriptionService) HandleAppleNotification(ctx context.Context, req *v1pb.HandleAppleNotificationRequest) (*emptypb.Empty, error) {
	handler := NewAppleNotificationHandler(s.store)
	err := handler.HandleNotification(ctx, req.SignedPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle notification: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *SubscriptionService) ListSubscriptionHistory(ctx context.Context, req *v1pb.ListSubscriptionHistoryRequest) (*v1pb.ListSubscriptionHistoryResponse, error) {
	userID, err := extractOrGetCurrentUserID(ctx, req.Parent)
	if err != nil {
		return nil, err
	}

	limit := 50
	if req.PageSize > 0 {
		limit = int(req.PageSize)
	}

	history, err := s.store.ListSubscriptionHistory(ctx, &store.FindSubscriptionHistory{
		UserID: &userID,
		Limit:  &limit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list subscription history: %v", err)
	}

	events := make([]*v1pb.SubscriptionHistoryEvent, len(history))
	for i, h := range history {
		events[i] = &v1pb.SubscriptionHistoryEvent{
			EventType:  convertEventTypeToProto(h.EventType),
			EventTime:  convertTimestamp(h.EventTs),
			ProductId:  ptrToString(h.ProductID),
			FromStatus: ptrToString(h.FromStatus),
			ToStatus:   ptrToString(h.ToStatus),
		}
	}

	return &v1pb.ListSubscriptionHistoryResponse{
		Events:     events,
		TotalSize:  int32(len(events)),
	}, nil
}

// Apple收据验证相关

const (
	AppleProductionVerifyURL = "https://buy.itunes.apple.com/verifyReceipt"
	AppleSandboxVerifyURL    = "https://sandbox.itunes.apple.com/verifyReceipt"
)

type AppleReceiptRequest struct {
	ReceiptData            string `json:"receipt-data"`
	Password               string `json:"password,omitempty"`
	ExcludeOldTransactions bool   `json:"exclude-old-transactions,omitempty"`
}

type AppleReceiptResponse struct {
	Status            int    `json:"status"`
	Environment       string `json:"environment"`
	Receipt           Receipt `json:"receipt"`
	LatestReceiptInfo []LatestReceiptInfo `json:"latest_receipt_info,omitempty"`
	LatestReceipt     string `json:"latest_receipt,omitempty"`
	Retryable         bool   `json:"is-retryable,omitempty"`
}

type Receipt struct {
	BundleID                 string `json:"bundle_id"`
	ApplicationVersion       string `json:"application_version"`
	OriginalApplicationVersion string `json:"original_application_version"`
	ReceiptCreationDate      string `json:"receipt_creation_date"`
	ExpirationDate           string `json:"expiration_date,omitempty"`
	InApp                    []InAppReceipt `json:"in_app,omitempty"`
}

type InAppReceipt struct {
	Quantity               string `json:"quantity"`
	ProductID              string `json:"product_id"`
	TransactionID          string `json:"transaction_id"`
	OriginalTransactionID  string `json:"original_transaction_id"`
	PurchaseDate           string `json:"purchase_date"`
	PurchaseDateMs         string `json:"purchase_date_ms"`
	PurchaseDatePst        string `json:"purchase_date_pst"`
	OriginalPurchaseDate   string `json:"original_purchase_date"`
	OriginalPurchaseDateMs string `json:"original_purchase_date_ms"`
	OriginalPurchaseDatePst string `json:"original_purchase_date_pst"`
	ExpiresDate            string `json:"expires_date,omitempty"`
	ExpiresDateMs          string `json:"expires_date_ms,omitempty"`
	ExpiresDatePst         string `json:"expires_date_pst,omitempty"`
	IsTrialPeriod          string `json:"is_trial_period"`
	IsInIntroOfferPeriod   string `json:"is_in_intro_offer_period"`
	WebOrderLineItemID     string `json:"web_order_line_item_id,omitempty"`
}

type LatestReceiptInfo struct {
	Quantity               string `json:"quantity"`
	ProductID              string `json:"product_id"`
	TransactionID          string `json:"transaction_id"`
	OriginalTransactionID  string `json:"original_transaction_id"`
	PurchaseDate           string `json:"purchase_date"`
	PurchaseDateMs         string `json:"purchase_date_ms"`
	PurchaseDatePst        string `json:"purchase_date_pst"`
	OriginalPurchaseDate   string `json:"original_purchase_date"`
	OriginalPurchaseDateMs string `json:"original_purchase_date_ms"`
	OriginalPurchaseDatePst string `json:"original_purchase_date_pst"`
	ExpiresDate            string `json:"expires_date,omitempty"`
	ExpiresDateMs          string `json:"expires_date_ms,omitempty"`
	ExpiresDatePst         string `json:"expires_date_pst,omitempty"`
	IsTrialPeriod          string `json:"is_trial_period"`
	IsInIntroOfferPeriod   string `json:"is_in_intro_offer_period"`
	WebOrderLineItemID     string `json:"web_order_line_item_id,omitempty"`
	CancellationDate       string `json:"cancellation_date,omitempty"`
	CancellationDateMs     string `json:"cancellation_date_ms,omitempty"`
	CancellationDatePst    string `json:"cancellation_date_pst,omitempty"`
	CancellationReason     string `json:"cancellation_reason,omitempty"`
}

func (s *SubscriptionService) validateReceiptWithApple(ctx context.Context, receiptData string, sandbox bool) (*AppleReceiptResponse, error) {
	verifyURL := AppleProductionVerifyURL
	if sandbox {
		verifyURL = AppleSandboxVerifyURL
	}

	requestBody := AppleReceiptRequest{
		ReceiptData:            receiptData,
		ExcludeOldTransactions: true,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal receipt request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", verifyURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to Apple")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var receiptResp AppleReceiptResponse
	if err := json.Unmarshal(body, &receiptResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	// 如果生产环境返回21007，说明这是沙盒收据，需要重新验证
	if receiptResp.Status == 21007 && !sandbox {
		return s.validateReceiptWithApple(ctx, receiptData, true)
	}

	// 检查状态码
	if receiptResp.Status != 0 {
		return nil, fmt.Errorf("receipt validation failed with status: %d", receiptResp.Status)
	}

	return &receiptResp, nil
}

func (s *SubscriptionService) processValidatedReceipt(ctx context.Context, userID int32, receiptResp *AppleReceiptResponse) (*store.UserSubscription, error) {
	if len(receiptResp.LatestReceiptInfo) == 0 {
		return nil, errors.New("no subscription found in receipt")
	}

	// 获取最新的收据信息
	latestReceipt := receiptResp.LatestReceiptInfo[0]
	for _, r := range receiptResp.LatestReceiptInfo {
		if r.ExpiresDateMs > latestReceipt.ExpiresDateMs {
			latestReceipt = r
		}
	}

	// 解析时间戳
	purchaseDateTs, err := parseAppleTimestamp(latestReceipt.PurchaseDateMs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse purchase date")
	}

	expiresDateTs, err := parseAppleTimestamp(latestReceipt.ExpiresDateMs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse expires date")
	}

	// 检查是否已存在订阅
	existingSub, err := s.store.GetUserSubscription(ctx, &store.FindUserSubscription{
		OriginalTransactionID: &latestReceipt.OriginalTransactionID,
	})
	if err != nil && !errors.Is(err, errors.New("subscription not found")) {
		return nil, errors.Wrap(err, "failed to check existing subscription")
	}

	var subscription *store.UserSubscription
	if existingSub != nil {
		// 更新现有订阅
		subscription, err = s.store.UpdateUserSubscription(ctx, &store.UpdateUserSubscription{
			ID:            existingSub.ID,
			TransactionID: &latestReceipt.TransactionID,
			Status:        ptrSubscriptionStatus(store.SubscriptionStatusActive),
			ExpiresDateTs: &expiresDateTs,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to update subscription")
		}
	} else {
		// 创建新订阅
		isTrial := latestReceipt.IsTrialPeriod == "true"
		env := "PRODUCTION"
		if receiptResp.Environment == "Sandbox" {
			env = "SANDBOX"
		}

		subscription, err = s.store.CreateUserSubscription(ctx, &store.UserSubscription{
			UserID:                userID,
			OriginalTransactionID: latestReceipt.OriginalTransactionID,
			TransactionID:         latestReceipt.TransactionID,
			ProductID:             latestReceipt.ProductID,
			Status:                store.SubscriptionStatusActive,
			PurchaseDateTs:        purchaseDateTs,
			ExpiresDateTs:         expiresDateTs,
			IsTrialPeriod:         isTrial,
			Environment:           env,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create subscription")
		}
	}

	// 更新VIP状态
	_, err = s.store.UpdateUserVIPStatus(ctx, &store.UpdateUserVIPStatus{
		UserID:         userID,
		IsVIP:          ptrBool(true),
		VipType:        ptrVipType(store.VipTypeSubscription),
		SubscriptionID: ptrInt32(subscription.ID),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update VIP status")
	}

	// 更新存储配额
	_, err = s.store.UpdateUserStorageUsage(ctx, &store.UpdateUserStorageUsage{
		UserID:     userID,
		QuotaBytes: ptrInt64(store.VIPQuotaBytes),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update storage quota")
	}

	// 记录订阅历史
	eventType := "PURCHASE"
	if existingSub != nil {
		eventType = "RENEWAL"
	}
	_, err = s.store.CreateSubscriptionHistory(ctx, &store.SubscriptionHistory{
		UserID:                userID,
		EventType:             eventType,
		EventTs:               time.Now().Unix(),
		OriginalTransactionID: &latestReceipt.OriginalTransactionID,
		ProductID:             &latestReceipt.ProductID,
		ToStatus:              ptrString("ACTIVE"),
	})
	if err != nil {
		// 记录失败不影响主流程
		fmt.Printf("failed to create subscription history: %v\n", err)
	}

	return subscription, nil
}

func parseAppleTimestamp(timestampMs string) (int64, error) {
	var ms int64
	_, err := fmt.Sscanf(timestampMs, "%d", &ms)
	if err != nil {
		return 0, err
	}
	return ms / 1000, nil
}

func convertVipTypeToProto(vipType store.VipType) v1pb.VipType {
	switch vipType {
	case store.VipTypeNone:
		return v1pb.VipType_NONE
	case store.VipTypeTrial:
		return v1pb.VipType_TRIAL
	case store.VipTypeSubscription:
		return v1pb.VipType_SUBSCRIPTION
	default:
		return v1pb.VipType_VIP_TYPE_UNSPECIFIED
	}
}

func convertSubscriptionStatusToProto(status store.SubscriptionStatus) v1pb.SubscriptionState {
	switch status {
	case store.SubscriptionStatusActive:
		return v1pb.SubscriptionState_ACTIVE
	case store.SubscriptionStatusExpired:
		return v1pb.SubscriptionState_EXPIRED
	case store.SubscriptionStatusCancelled:
		return v1pb.SubscriptionState_CANCELLED
	case store.SubscriptionStatusGracePeriod:
		return v1pb.SubscriptionState_GRACE_PERIOD
	case store.SubscriptionStatusBillingRetry:
		return v1pb.SubscriptionState_BILLING_RETRY
	default:
		return v1pb.SubscriptionState_SUBSCRIPTION_STATE_UNSPECIFIED
	}
}

func convertEventTypeToProto(eventType string) v1pb.SubscriptionHistoryEvent_EventType {
	switch eventType {
	case "PURCHASE":
		return v1pb.SubscriptionHistoryEvent_PURCHASE
	case "RENEWAL":
		return v1pb.SubscriptionHistoryEvent_RENEWAL
	case "CANCEL":
		return v1pb.SubscriptionHistoryEvent_CANCEL
	case "EXPIRE":
		return v1pb.SubscriptionHistoryEvent_EXPIRE
	case "REFUND":
		return v1pb.SubscriptionHistoryEvent_REFUND
	case "TRIAL_START":
		return v1pb.SubscriptionHistoryEvent_TRIAL_START
	case "TRIAL_END":
		return v1pb.SubscriptionHistoryEvent_TRIAL_END
	default:
		return v1pb.SubscriptionHistoryEvent_EVENT_TYPE_UNSPECIFIED
	}
}

func convertTimestamp(ts int64) *timestamppb.Timestamp {
	return timestamppb.New(time.Unix(ts, 0))
}

func ptrBool(v bool) *bool {
	return &v
}

func ptrInt32(v int32) *int32 {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrString(v string) *string {
	return &v
}

func ptrVipType(v store.VipType) *store.VipType {
	return &v
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

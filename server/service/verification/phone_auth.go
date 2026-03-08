package verification

import (
	"context"
	"fmt"
	"time"

	"github.com/usememos/memos/store"
)

// PhoneAuthService 号码认证服务
type PhoneAuthService struct {
	store *store.Store
}

// NewPhoneAuthService 创建号码认证服务
func NewPhoneAuthService(store *store.Store) *PhoneAuthService {
	return &PhoneAuthService{
		store: store,
	}
}

// VerifyPhone 验证手机号
func (s *PhoneAuthService) VerifyPhone(ctx context.Context, phoneNumber, authToken, purpose string) (string, error) {
	// 这里实现号码认证逻辑，调用第三方服务验证 authToken
	// 示例：验证 authToken 是否有效
	
	// 生成验证ID
	verificationID := fmt.Sprintf("phone_auth_%d", time.Now().UnixNano())
	
	// 保存验证记录
	expiresTs := time.Now().Add(10 * time.Minute).Unix()
	_, err := s.store.CreateVerification(ctx, &store.Verification{
		PhoneNumber: phoneNumber,
		Method:      "PHONE_AUTH",
		Purpose:     purpose,
		Code:        verificationID,
		CreatedTs:   time.Now().Unix(),
		ExpiresTs:   expiresTs,
		IsUsed:      false,
	})
	
	return verificationID, err
}

// SendSMSCode 发送短信验证码（号码认证服务不需要实现）
func (s *PhoneAuthService) SendSMSCode(ctx context.Context, phoneNumber, purpose string) (string, error) {
	return "", fmt.Errorf("sms service not enabled")
}

// VerifySMSCode 验证短信验证码（号码认证服务不需要实现）
func (s *PhoneAuthService) VerifySMSCode(ctx context.Context, phoneNumber, code, purpose string) (bool, error) {
	return false, fmt.Errorf("sms service not enabled")
}

// CheckSMSCode 检查短信验证码是否有效（号码认证服务不需要实现）
func (s *PhoneAuthService) CheckSMSCode(ctx context.Context, phoneNumber, code, purpose string) (bool, error) {
	return false, fmt.Errorf("sms service not enabled")
}
package verification

import (
	"context"
)

// Service 验证服务接口
type Service interface {
	// SendSMSCode 发送短信验证码
	SendSMSCode(ctx context.Context, phoneNumber, purpose string) (string, error)
	
	// VerifySMSCode 验证短信验证码
	VerifySMSCode(ctx context.Context, phoneNumber, code, purpose string) (bool, error)
	
	// CheckSMSCode 检查短信验证码是否有效（不标记为已使用）
	CheckSMSCode(ctx context.Context, phoneNumber, code, purpose string) (bool, error)
	
	// VerifyPhone 验证手机号（号码认证）
	VerifyPhone(ctx context.Context, phoneNumber, authToken, purpose string) (string, error)
}
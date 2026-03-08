package store

import (
	"context"
)

// Verification 验证记录
type Verification struct {
	ID          int32
	PhoneNumber string
	Method      string
	Purpose     string
	Code        string
	CreatedTs   int64
	ExpiresTs   int64
	IsUsed      bool
}

// CreateVerification 创建验证记录
func (s *Store) CreateVerification(ctx context.Context, verification *Verification) (*Verification, error) {
	return s.driver.CreateVerification(ctx, verification)
}

// GetVerification 获取验证记录
func (s *Store) GetVerification(ctx context.Context, phoneNumber, code, purpose string) (*Verification, error) {
	return s.driver.GetVerification(ctx, phoneNumber, code, purpose)
}

// UpdateVerification 更新验证记录
func (s *Store) UpdateVerification(ctx context.Context, verification *Verification) error {
	return s.driver.UpdateVerification(ctx, verification)
}
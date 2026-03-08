package verification

import (
	"context"
	"fmt"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dypnsapi20170525 "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/pkg/errors"

	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

type SMSService struct {
	store *store.Store
}

func NewSMSService(store *store.Store) *SMSService {
	return &SMSService{
		store: store,
	}
}

func (s *SMSService) getSMSSetting(ctx context.Context) (*storepb.InstanceSmsSetting, error) {
	smsSetting, err := s.store.GetInstanceSmsSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SMS setting")
	}
	if smsSetting == nil || smsSetting.ApiKey == "" || smsSetting.ApiSecret == "" {
		return nil, errors.New("SMS service not configured")
	}
	return smsSetting, nil
}

func (s *SMSService) createClient(smsSetting *storepb.InstanceSmsSetting) (*dypnsapi20170525.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(smsSetting.ApiKey),
		AccessKeySecret: tea.String(smsSetting.ApiSecret),
	}
	endpoint := smsSetting.Endpoint
	if endpoint == "" {
		endpoint = "dypnsapi.aliyuncs.com"
	}
	config.Endpoint = tea.String(endpoint)

	client, err := dypnsapi20170525.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	return client, nil
}

func (s *SMSService) SendSMSCode(ctx context.Context, phoneNumber, purpose string) (string, error) {
	smsSetting, err := s.getSMSSetting(ctx)
	if err != nil {
		return "", err
	}

	code, err := s.sendSMSViaAliyun(smsSetting, phoneNumber)
	if err != nil {
		return "", errors.Wrap(err, "failed to send SMS")
	}

	expiresTs := time.Now().Add(10 * time.Minute).Unix()
	_, err = s.store.CreateVerification(ctx, &store.Verification{
		PhoneNumber: phoneNumber,
		Method:      "SMS",
		Purpose:     purpose,
		Code:        code,
		CreatedTs:   time.Now().Unix(),
		ExpiresTs:   expiresTs,
		IsUsed:      false,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create verification record")
	}

	return code, nil
}

func (s *SMSService) VerifySMSCode(ctx context.Context, phoneNumber, code, purpose string) (bool, error) {
	verification, err := s.store.GetVerification(ctx, phoneNumber, code, purpose)
	if err != nil {
		return false, err
	}

	if verification == nil {
		return false, nil
	}

	if verification.IsUsed {
		return false, nil
	}

	if time.Now().Unix() > verification.ExpiresTs {
		return false, nil
	}

	err = s.store.UpdateVerification(ctx, &store.Verification{
		ID:     verification.ID,
		IsUsed: true,
	})

	return true, err
}

func (s *SMSService) CheckSMSCode(ctx context.Context, phoneNumber, code, purpose string) (bool, error) {
	verification, err := s.store.GetVerification(ctx, phoneNumber, code, purpose)
	if err != nil {
		return false, err
	}

	if verification == nil {
		return false, nil
	}

	if verification.IsUsed {
		return false, nil
	}

	if time.Now().Unix() > verification.ExpiresTs {
		return false, nil
	}

	return true, nil
}

func (s *SMSService) VerifyPhone(ctx context.Context, phoneNumber, authToken, purpose string) (string, error) {
	return "", fmt.Errorf("phone auth not supported by SMS service")
}

func (s *SMSService) sendSMSViaAliyun(smsSetting *storepb.InstanceSmsSetting, phoneNumber string) (string, error) {
	client, err := s.createClient(smsSetting)
	if err != nil {
		return "", err
	}

	signName := smsSetting.SignName
	if signName == "" {
		signName = "速通互联验证码"
	}

	sendSmsVerifyCodeRequest := &dypnsapi20170525.SendSmsVerifyCodeRequest{
		SignName:     tea.String(signName),
		TemplateCode: tea.String(smsSetting.TemplateId),
		PhoneNumber:  tea.String(phoneNumber),
		TemplateParam: tea.String(`{"code":"##code##","min":"10"}`),
		CodeType:     tea.Int64(1),
		CodeLength:   tea.Int64(6),
		ValidTime:    tea.Int64(600),
		ReturnVerifyCode: tea.Bool(true),
	}

	runtime := &service.RuntimeOptions{}

	var verifyCode string

	tryErr := func() error {
		resp, err := client.SendSmsVerifyCodeWithOptions(sendSmsVerifyCodeRequest, runtime)
		if err != nil {
			return err
		}

		if resp.Body == nil || resp.Body.Code == nil {
			return errors.New("empty response from SMS API")
		}

		if *resp.Body.Code != "OK" {
			message := ""
			if resp.Body.Message != nil {
				message = *resp.Body.Message
			}
			return fmt.Errorf("SMS API error: %s - %s", *resp.Body.Code, message)
		}

		if resp.Body.Model != nil && resp.Body.Model.VerifyCode != nil {
			verifyCode = *resp.Body.Model.VerifyCode
		}

		return nil
	}()

	if tryErr != nil {
		return "", tryErr
	}

	if verifyCode == "" {
		return "", errors.New("failed to get verification code from SMS API response")
	}

	return verifyCode, nil
}

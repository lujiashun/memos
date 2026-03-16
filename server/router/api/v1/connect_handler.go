package v1

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/proto/gen/api/v1/apiv1connect"
)

// ConnectServiceHandler wraps APIV1Service to implement Connect handler interfaces.
// It adapts the existing gRPC service implementations to work with Connect's
// request/response wrapper types.
//
// This wrapper pattern allows us to:
// - Reuse existing gRPC service implementations
// - Support both native gRPC and Connect protocols
// - Maintain a single source of truth for business logic.
type ConnectServiceHandler struct {
	*APIV1Service
}

// NewConnectServiceHandler creates a new Connect service handler.
func NewConnectServiceHandler(svc *APIV1Service) *ConnectServiceHandler {
	return &ConnectServiceHandler{APIV1Service: svc}
}

// RegisterConnectHandlers registers all Connect service handlers on the given mux.
func (s *ConnectServiceHandler) RegisterConnectHandlers(mux *http.ServeMux, opts ...connect.HandlerOption) {
	// Register all service handlers
	handlers := []struct {
		path    string
		handler http.Handler
	}{
		wrap(apiv1connect.NewInstanceServiceHandler(s, opts...)),
		wrap(apiv1connect.NewAuthServiceHandler(s, opts...)),
		wrap(apiv1connect.NewUserServiceHandler(s, opts...)),
		wrap(apiv1connect.NewMemoServiceHandler(s, opts...)),
		wrap(apiv1connect.NewAttachmentServiceHandler(s, opts...)),
		wrap(apiv1connect.NewShortcutServiceHandler(s, opts...)),
		wrap(apiv1connect.NewActivityServiceHandler(s, opts...)),
		wrap(apiv1connect.NewIdentityProviderServiceHandler(s, opts...)),
	}

	for _, h := range handlers {
		mux.Handle(h.path, h.handler)
	}
}

// wrap converts (path, handler) return value to a struct for cleaner iteration.
func wrap(path string, handler http.Handler) struct {
	path    string
	handler http.Handler
} {
	return struct {
		path    string
		handler http.Handler
	}{path, handler}
}

// convertGRPCError converts gRPC status errors to Connect errors.
// This preserves the error code semantics between the two protocols.
func convertGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return connect.NewError(grpcCodeToConnectCode(st.Code()), err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// grpcCodeToConnectCode converts gRPC status codes to Connect error codes.
// gRPC and Connect use the same error code semantics, so this is a direct cast.
// See: https://connectrpc.com/docs/protocol/#error-codes
func grpcCodeToConnectCode(code codes.Code) connect.Code {
	return connect.Code(code)
}

// SendVerificationCode implements apiv1connect.AuthServiceHandler
func (s *ConnectServiceHandler) SendVerificationCode(ctx context.Context, req *connect.Request[v1pb.SendVerificationCodeRequest]) (*connect.Response[v1pb.SendVerificationCodeResponse], error) {
	resp, err := s.APIV1Service.SendVerificationCode(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// VerifyPhone implements apiv1connect.AuthServiceHandler
func (s *ConnectServiceHandler) VerifyPhone(ctx context.Context, req *connect.Request[v1pb.VerifyPhoneRequest]) (*connect.Response[v1pb.VerifyPhoneResponse], error) {
	resp, err := s.APIV1Service.VerifyPhone(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// ResetPassword implements apiv1connect.AuthServiceHandler
func (s *ConnectServiceHandler) ResetPassword(ctx context.Context, req *connect.Request[v1pb.ResetPasswordRequest]) (*connect.Response[v1pb.ResetPasswordResponse], error) {
	resp, err := s.APIV1Service.ResetPassword(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// BatchArchiveUsers implements apiv1connect.UserServiceHandler
func (s *ConnectServiceHandler) BatchArchiveUsers(ctx context.Context, req *connect.Request[v1pb.BatchArchiveUsersRequest]) (*connect.Response[v1pb.BatchArchiveUsersResponse], error) {
	resp, err := s.APIV1Service.BatchArchiveUsers(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// BatchRestoreUsers implements apiv1connect.UserServiceHandler
func (s *ConnectServiceHandler) BatchRestoreUsers(ctx context.Context, req *connect.Request[v1pb.BatchRestoreUsersRequest]) (*connect.Response[v1pb.BatchRestoreUsersResponse], error) {
	resp, err := s.APIV1Service.BatchRestoreUsers(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// BatchDeleteUsers implements apiv1connect.UserServiceHandler
func (s *ConnectServiceHandler) BatchDeleteUsers(ctx context.Context, req *connect.Request[v1pb.BatchDeleteUsersRequest]) (*connect.Response[v1pb.BatchDeleteUsersResponse], error) {
	resp, err := s.APIV1Service.BatchDeleteUsers(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

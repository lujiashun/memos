# Rate Limiting in Memos

## Overview

This document describes the rate limiting mechanism implemented in Memos to prevent abuse and ensure system stability, particularly for memo creation requests.

## Implementation Details

### Rate Limiter Component

The rate limiting functionality is implemented through the following components:

1. **RateLimitConfig**: Configuration struct that defines rate limit settings
2. **RateLimiter**: Core rate limiting logic that tracks request counts
3. **RateLimitInterceptor**: Connect RPC interceptor for rate limiting
4. **Global Rate Limiter**: Shared instance used across the application

### Files Modified/Added

- `server/router/api/v1/rate_limit_interceptor.go`: Core rate limiting implementation
- `server/router/api/v1/memo_service.go`: Rate limit check in CreateMemo method
- `server/router/api/v1/v1.go`: Interceptor registration for Connect and gRPC-Gateway

## Rate Limit Configuration

### Default Settings

The default rate limit configuration is:

```go
var DefaultRateLimitConfig = RateLimitConfig{
    UserLimit:  60,  // 60 requests per minute per user
    IPLimit:    100, // 100 requests per minute per IP
    WindowSize: time.Minute,
}
```

### Configuration Options

- **UserLimit**: Maximum number of requests per user per time window
- **IPLimit**: Maximum number of requests per IP address per time window
- **WindowSize**: Time window for rate limiting (e.g., 1 minute)

## How It Works

### Request Tracking

The rate limiter tracks requests using two maps:

1. **userCounts**: Maps user IDs to timestamps of their requests
2. **ipCounts**: Maps IP addresses to timestamps of their requests

### Cleanup Mechanism

A background goroutine periodically cleans up old entries to prevent memory leaks:

- Runs every `WindowSize` interval
- Removes entries older than the current time minus `WindowSize`

### Rate Limit Check

When a CreateMemo request is received:

1. Extract the user ID (if authenticated)
2. Extract the client IP address from headers (X-Forwarded-For, X-Real-IP, or fallback to "unknown")
3. Check if the user has exceeded their limit
4. Check if the IP has exceeded its limit
5. If either limit is exceeded, return a `ResourceExhausted` error
6. If within limits, record the request and allow it to proceed

## Integration Points

### Connect RPC

The rate limit interceptor is added to the Connect RPC interceptor chain:

```go
connectInterceptors := connect.WithInterceptors(
    NewMetadataInterceptor(),
    NewLoggingInterceptor(logStacktraces),
    NewRecoveryInterceptor(logStacktraces),
    NewRateLimitInterceptor(DefaultRateLimitConfig),
    NewAuthInterceptor(s.Store, s.Secret),
)
```

### gRPC-Gateway (REST API)

A rate limit middleware is added to the gRPC-Gateway mux:

```go
gwMux := runtime.NewServeMux(
    runtime.WithMiddlewares(gatewayRateLimitMiddleware, gatewayAuthMiddleware),
)
```

### MemoService.CreateMemo

An additional rate limit check is implemented directly in the CreateMemo method to ensure all request paths are covered:

```go
// Check rate limit
userID := int32(0)
if user != nil {
    userID = user.ID
}
if !globalRateLimiter.Allow(userID, ip) {
    return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded: too many memo creation requests")
}
```

## Error Response

When rate limit is exceeded, the API returns a `ResourceExhausted` error with the message:

```json
{
  "code": 8,
  "message": "rate limit exceeded: too many memo creation requests",
  "details": []
}
```

## Testing

### Manual Testing

To test the rate limiting functionality:

1. Set a low IP limit for testing (e.g., 3 requests per minute)
2. Send multiple CreateMemo requests
3. Observe that the 4th request returns a rate limit error
4. Wait for the time window to expire
5. Send another request and verify it succeeds

### Example Test Command

```bash
for i in {1..4}; do \
  curl -X POST http://localhost:8081/api/v1/memos \
  -H "Content-Type: application/json" \
  -d '{"memo": {"content": "test memo "}}' \
  2>&1 | grep -E '(rate limit|Too Many Requests|code)'; \
done
```

## Best Practices

### For Developers

1. **Adjust Limits**: Modify the default rate limit values based on your deployment scale and expected usage
2. **Monitor**: Keep an eye on rate limit errors in logs to identify potential abuse
3. **Tune**: Adjust window size and limits based on observed usage patterns

### For API Clients

1. **Implement Backoff**: If you receive a rate limit error, implement exponential backoff before retrying
2. **Batch Operations**: Combine multiple memo creations into fewer requests when possible
3. **Respect Limits**: Design your client to stay within the rate limits to avoid service disruptions

## Customization

To customize the rate limit settings:

1. Modify the `DefaultRateLimitConfig` in `rate_limit_interceptor.go`
2. Or create a new config and pass it to `NewRateLimitInterceptor()`

## Conclusion

The rate limiting mechanism provides an effective way to prevent abuse and ensure system stability by limiting the number of memo creation requests per user and per IP address. The implementation is flexible and can be adjusted based on specific deployment needs.
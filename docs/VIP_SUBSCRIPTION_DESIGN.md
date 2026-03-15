# Memos VIP订阅系统技术设计方案

> 版本: 1.0  
> 日期: 2026-03-10  
> 作者: AI Assistant

---

## 目录

1. [需求概述](#1-需求概述)
2. [系统架构](#2-系统架构)
3. [数据库设计](#3-数据库设计)
4. [后端API设计](#4-后端api设计)
5. [iOS前端设计](#5-ios前端设计)
6. [StoreKit集成](#6-storekit集成)
7. [Apple服务器通知处理](#7-apple服务器通知处理)
8. [业务流程](#8-业务流程)
9. [苹果审核合规](#9-苹果审核合规)
10. [测试方案](#10-测试方案)
11. [部署计划](#11-部署计划)

---

## 1. 需求概述

### 1.1 VIP权益

| 权益 | 普通用户 | VIP用户 |
|------|---------|---------|
| 备忘录数量 | 无限制 | 无限制 |
| 存储空间 | 50MB | 5GB |
| 广告 | 显示广告 | 无广告 |
| 高级功能 | - | 优先体验新功能 |

### 1.2 用户生命周期

```
新用户注册 → 10天VIP试用期 → 试用期结束 → 普通用户
                                    ↓
                            购买年度订阅 → VIP用户
                                    ↓
                            订阅到期/取消 → 普通用户（只读模式）
```

### 1.3 核心功能

- ✅ 新用户10天VIP试用期
- ✅ 年度订阅购买（iOS IAP）
- ✅ 订阅管理（取消、恢复购买、状态查询）
- ✅ 存储配额管理
- ✅ 多设备订阅状态同步
- ✅ 退款处理
- ✅ 广告系统（普通用户）

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        iOS Client (MoeMemos)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  StoreKit   │  │   SwiftUI   │  │  Subscription Manager   │  │
│  │  Framework  │  │    Views    │  │      (Local Cache)      │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
└─────────┼────────────────┼─────────────────────┼────────────────┘
          │                │                     │
          │ Apple App Store│                     │ Connect RPC
          │                │                     │
          ▼                │                     ▼
┌─────────────────┐        │        ┌─────────────────────────────┐
│  Apple Servers  │        │        │     Memos Backend (Go)      │
│  ┌───────────┐  │        │        │  ┌───────────────────────┐  │
│  │ App Store │  │        │        │  │  Subscription Service │  │
│  │   Connect │◄─┼────────┼────────┼──►  (Server-to-Server)  │  │
│  └───────────┘  │        │        │  └───────────────────────┘  │
│  ┌───────────┐  │        │        │  ┌───────────────────────┐  │
│  │   IAP     │  │        │        │  │   Storage Quota       │  │
│  │ Validation│  │        │        │  │      Manager          │  │
│  └───────────┘  │        │        │  └───────────────────────┘  │
└─────────────────┘        │        │  ┌───────────────────────┐  │
                           │        │  │   Ad System           │  │
                           │        │  │   (Future)            │  │
                           │        │  └───────────────────────┘  │
                           │        └─────────────────────────────┘
                           │                       │
                           │                       ▼
                           │        ┌─────────────────────────────┐
                           │        │      Database (SQLite)      │
                           │        │  ┌───────────────────────┐  │
                           └────────►  │   user_subscription   │  │
                                    │   user_storage_usage   │  │
                                    │   subscription_history │  │
                                    └───────────────────────┘  │
                                    └─────────────────────────────┘
```

### 2.2 数据流

#### 2.2.1 购买流程

```
用户点击订阅 → StoreKit显示购买界面 → 用户确认购买 → 
Apple处理支付 → 返回收据 → iOS客户端发送收据到后端 → 
后端验证收据 → 创建订阅记录 → 返回VIP状态
```

#### 2.2.2 订阅状态同步

```
iOS客户端启动 → 请求订阅状态API → 后端返回订阅信息 → 
本地缓存订阅状态 → 定期轮询更新（可选）
```

#### 2.2.3 Apple服务器通知

```
订阅状态变化 → Apple App Store Connect → 发送通知到后端Webhook → 
后端验证签名 → 更新订阅状态 → 用户下次同步时获取最新状态
```

---

## 3. 数据库设计

### 3.1 新增表结构

#### 3.1.1 用户订阅表 (user_subscription)

```sql
CREATE TABLE user_subscription (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    
    -- 订阅标识
    original_transaction_id TEXT NOT NULL,  -- Apple原始交易ID，唯一标识订阅
    transaction_id TEXT NOT NULL,           -- 当前交易ID（续费会变化）
    product_id TEXT NOT NULL,               -- 产品ID，如 "com.memos.vip.yearly"
    
    -- 订阅状态
    status TEXT NOT NULL,                   -- ACTIVE, EXPIRED, CANCELLED, GRACE_PERIOD, BILLING_RETRY
    
    -- 时间信息
    purchase_date_ts INTEGER NOT NULL,      -- 购买时间（Unix时间戳）
    expires_date_ts INTEGER NOT NULL,       -- 到期时间（Unix时间戳）
    cancellation_date_ts INTEGER,           -- 取消时间（如果取消）
    grace_period_expires_ts INTEGER,        -- 宽限期到期时间
    
    -- 试用信息
    is_trial_period BOOLEAN DEFAULT FALSE,  -- 是否试用期
    is_in_intro_offer BOOLEAN DEFAULT FALSE,-- 是否入门优惠
    
    -- Apple通知相关
    environment TEXT NOT NULL,              -- PRODUCTION 或 SANDBOX
    last_notification_type TEXT,            -- 最后收到的通知类型
    last_notification_ts INTEGER,           -- 最后通知时间
    
    -- 收据信息（加密存储）
    receipt_data TEXT,                      -- Base64编码的收据数据
    
    -- 标准字段
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    
    -- 约束
    UNIQUE(original_transaction_id),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_subscription_user_id ON user_subscription(user_id);
CREATE INDEX idx_user_subscription_status ON user_subscription(status);
CREATE INDEX idx_user_subscription_expires ON user_subscription(expires_date_ts);
```

#### 3.1.2 用户存储配额表 (user_storage_usage)

```sql
CREATE TABLE user_storage_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    
    -- 存储使用量（字节）
    total_bytes INTEGER NOT NULL DEFAULT 0,     -- 总使用量
    attachment_bytes INTEGER NOT NULL DEFAULT 0, -- 附件使用量
    memo_content_bytes INTEGER NOT NULL DEFAULT 0, -- 备忘录内容使用量
    
    -- 配额限制（字节）
    quota_bytes INTEGER NOT NULL DEFAULT 52428800, -- 默认50MB
    
    -- 统计时间
    last_calculated_ts INTEGER NOT NULL,
    
    -- 标准字段
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_storage_user_id ON user_storage_usage(user_id);
```

#### 3.1.3 订阅历史表 (subscription_history)

```sql
CREATE TABLE subscription_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    
    -- 事件信息
    event_type TEXT NOT NULL,              -- PURCHASE, RENEWAL, CANCEL, EXPIRE, REFUND, TRIAL_START, TRIAL_END
    event_ts INTEGER NOT NULL,
    
    -- 订阅信息快照
    original_transaction_id TEXT,
    product_id TEXT,
    
    -- 状态变更
    from_status TEXT,
    to_status TEXT,
    
    -- 详细信息
    details TEXT,                          -- JSON格式的详细信息
    
    -- 标准字段
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscription_history_user_id ON subscription_history(user_id);
CREATE INDEX idx_subscription_history_event_ts ON subscription_history(event_ts);
```

#### 3.1.4 用户VIP状态表 (user_vip_status)

```sql
CREATE TABLE user_vip_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    
    -- VIP状态
    is_vip BOOLEAN NOT NULL DEFAULT FALSE,
    vip_type TEXT NOT NULL,                -- TRIAL, SUBSCRIPTION, NONE
    
    -- 试用期相关
    trial_start_ts INTEGER,                -- 试用期开始时间
    trial_end_ts INTEGER,                  -- 试用期结束时间
    trial_used BOOLEAN DEFAULT FALSE,      -- 是否已使用过试用期
    
    -- 订阅关联
    subscription_id INTEGER,               -- 关联到user_subscription表
    
    -- 标准字段
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_id) REFERENCES user_subscription(id) ON DELETE SET NULL
);

CREATE INDEX idx_user_vip_status_user_id ON user_vip_status(user_id);
CREATE INDEX idx_user_vip_status_is_vip ON user_vip_status(is_vip);
```

### 3.2 迁移文件

创建迁移文件：`store/migration/sqlite/0.30/1__subscription_system.sql`

```sql
-- 创建用户订阅表
CREATE TABLE user_subscription (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    original_transaction_id TEXT NOT NULL,
    transaction_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    status TEXT NOT NULL,
    purchase_date_ts INTEGER NOT NULL,
    expires_date_ts INTEGER NOT NULL,
    cancellation_date_ts INTEGER,
    grace_period_expires_ts INTEGER,
    is_trial_period BOOLEAN DEFAULT FALSE,
    is_in_intro_offer BOOLEAN DEFAULT FALSE,
    environment TEXT NOT NULL,
    last_notification_type TEXT,
    last_notification_ts INTEGER,
    receipt_data TEXT,
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    UNIQUE(original_transaction_id),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_subscription_user_id ON user_subscription(user_id);
CREATE INDEX idx_user_subscription_status ON user_subscription(status);
CREATE INDEX idx_user_subscription_expires ON user_subscription(expires_date_ts);

-- 创建用户存储配额表
CREATE TABLE user_storage_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    total_bytes INTEGER NOT NULL DEFAULT 0,
    attachment_bytes INTEGER NOT NULL DEFAULT 0,
    memo_content_bytes INTEGER NOT NULL DEFAULT 0,
    quota_bytes INTEGER NOT NULL DEFAULT 52428800,
    last_calculated_ts INTEGER NOT NULL,
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_storage_user_id ON user_storage_usage(user_id);

-- 创建订阅历史表
CREATE TABLE subscription_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    event_ts INTEGER NOT NULL,
    original_transaction_id TEXT,
    product_id TEXT,
    from_status TEXT,
    to_status TEXT,
    details TEXT,
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscription_history_user_id ON subscription_history(user_id);
CREATE INDEX idx_subscription_history_event_ts ON subscription_history(event_ts);

-- 创建用户VIP状态表
CREATE TABLE user_vip_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    is_vip BOOLEAN NOT NULL DEFAULT FALSE,
    vip_type TEXT NOT NULL DEFAULT 'NONE',
    trial_start_ts INTEGER,
    trial_end_ts INTEGER,
    trial_used BOOLEAN DEFAULT FALSE,
    subscription_id INTEGER,
    created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_id) REFERENCES user_subscription(id) ON DELETE SET NULL
);

CREATE INDEX idx_user_vip_status_user_id ON user_vip_status(user_id);
CREATE INDEX idx_user_vip_status_is_vip ON user_vip_status(is_vip);

-- 为现有用户初始化VIP状态
INSERT INTO user_vip_status (user_id, is_vip, vip_type, trial_used)
SELECT id, FALSE, 'NONE', FALSE FROM user WHERE row_status = 'NORMAL';

-- 为现有用户初始化存储配额
INSERT INTO user_storage_usage (user_id, total_bytes, quota_bytes, last_calculated_ts)
SELECT id, 0, 52428800, strftime('%s', 'now') FROM user WHERE row_status = 'NORMAL';
```

---

## 4. 后端API设计

### 4.1 Protocol Buffers定义

创建新文件：`proto/api/v1/subscription_service.proto`

```protobuf
syntax = "proto3";

package memos.api.v1;

import "api/v1/common.proto";
import "google/api/annotations.proto";
import "google/api/client.proto";
import "google/api/field_behavior.proto";
import "google/api/resource.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";

option go_package = "gen/api/v1";

// SubscriptionService manages user subscriptions and VIP status.
service SubscriptionService {
  // GetSubscriptionStatus returns the current subscription status for a user.
  rpc GetSubscriptionStatus(GetSubscriptionStatusRequest) returns (SubscriptionStatus) {
    option (google.api.http) = {get: "/api/v1/{name=users/*}/subscription"};
    option (google.api.method_signature) = "name";
  }

  // ValidateReceipt validates an Apple IAP receipt and creates/updates subscription.
  rpc ValidateReceipt(ValidateReceiptRequest) returns (ValidateReceiptResponse) {
    option (google.api.http) = {
      post: "/api/v1/{parent=users/*}/subscription:validateReceipt"
      body: "*"
    };
  }

  // RestorePurchases restores previous purchases for a user.
  rpc RestorePurchases(RestorePurchasesRequest) returns (RestorePurchasesResponse) {
    option (google.api.http) = {
      post: "/api/v1/{parent=users/*}/subscription:restore"
      body: "*"
    };
  }

  // GetStorageUsage returns the storage usage for a user.
  rpc GetStorageUsage(GetStorageUsageRequest) returns (StorageUsage) {
    option (google.api.http) = {get: "/api/v1/{name=users/*}/storage"};
    option (google.api.method_signature) = "name";
  }

  // HandleAppleNotification handles server-to-server notifications from Apple.
  // This is an internal endpoint called by Apple's servers.
  rpc HandleAppleNotification(HandleAppleNotificationRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/api/v1/apple/notifications"
      body: "*"
    };
  }

  // ListSubscriptionHistory returns the subscription history for a user.
  rpc ListSubscriptionHistory(ListSubscriptionHistoryRequest) returns (ListSubscriptionHistoryResponse) {
    option (google.api.http) = {get: "/api/v1/{parent=users/*}/subscriptionHistory"};
    option (google.api.method_signature) = "parent";
  }
}

// SubscriptionStatus represents the current subscription status.
message SubscriptionStatus {
  option (google.api.resource) = {
    type: "memos.api.v1/SubscriptionStatus"
    pattern: "users/{user}/subscription"
  };

  // The resource name of the subscription.
  // Format: users/{user}/subscription
  string name = 1 [(google.api.field_behavior) = IDENTIFIER];

  // Whether the user is a VIP.
  bool is_vip = 2;

  // The type of VIP status.
  VipType vip_type = 3;

  // The subscription details (if any).
  Subscription subscription = 4;

  // Trial information.
  TrialInfo trial_info = 5;

  // The storage quota in bytes.
  int64 storage_quota_bytes = 6;

  // The storage used in bytes.
  int64 storage_used_bytes = 7;

  // Whether the user has exceeded their storage quota.
  bool storage_exceeded = 8;
}

// VipType represents the type of VIP status.
enum VipType {
  VIP_TYPE_UNSPECIFIED = 0;
  // No VIP status.
  NONE = 1;
  // VIP from trial period.
  TRIAL = 2;
  // VIP from paid subscription.
  SUBSCRIPTION = 3;
}

// Subscription represents an active subscription.
message Subscription {
  // The product ID (e.g., "com.memos.vip.yearly").
  string product_id = 1;

  // The subscription status.
  SubscriptionState state = 2;

  // The purchase date.
  google.protobuf.Timestamp purchase_date = 3;

  // The expiration date.
  google.protobuf.Timestamp expires_date = 4;

  // Whether this is a trial subscription.
  bool is_trial = 5;

  // Whether the subscription will auto-renew.
  bool will_renew = 6;

  // The original transaction ID (unique identifier for the subscription).
  string original_transaction_id = 7;
}

// SubscriptionState represents the state of a subscription.
enum SubscriptionState {
  SUBSCRIPTION_STATE_UNSPECIFIED = 0;
  // Subscription is active.
  ACTIVE = 1;
  // Subscription has expired.
  EXPIRED = 2;
  // Subscription is in grace period (payment issue).
  GRACE_PERIOD = 3;
  // Subscription is in billing retry period.
  BILLING_RETRY = 4;
  // Subscription was cancelled by user.
  CANCELLED = 5;
}

// TrialInfo contains information about the trial period.
message TrialInfo {
  // Whether the user has used their trial.
  bool trial_used = 1;

  // The trial start date (if active).
  google.protobuf.Timestamp trial_start_date = 2;

  // The trial end date (if active).
  google.protobuf.Timestamp trial_end_date = 3;

  // Days remaining in trial (if active).
  int32 days_remaining = 4;
}

// StorageUsage represents storage usage information.
message StorageUsage {
  option (google.api.resource) = {
    type: "memos.api.v1/StorageUsage"
    pattern: "users/{user}/storage"
  };

  // The resource name.
  // Format: users/{user}/storage
  string name = 1 [(google.api.field_behavior) = IDENTIFIER];

  // Total bytes used.
  int64 used_bytes = 2;

  // Total bytes allowed (quota).
  int64 quota_bytes = 3;

  // Percentage used (0-100).
  int32 used_percentage = 4;

  // Breakdown by type.
  StorageBreakdown breakdown = 5;

  // Whether quota is exceeded.
  bool quota_exceeded = 6;
}

// StorageBreakdown shows storage usage by category.
message StorageBreakdown {
  // Bytes used by attachments.
  int64 attachment_bytes = 1;

  // Bytes used by memo content.
  int64 memo_content_bytes = 2;
}

message GetSubscriptionStatusRequest {
  // The resource name of the user.
  // Format: users/{user}
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "memos.api.v1/User"}
  ];
}

message ValidateReceiptRequest {
  // The parent user resource.
  // Format: users/{user}
  string parent = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "memos.api.v1/User"}
  ];

  // The Base64-encoded receipt data from Apple.
  string receipt_data = 2 [(google.api.field_behavior) = REQUIRED];

  // Whether this is a sandbox receipt.
  bool sandbox = 3 [(google.api.field_behavior) = OPTIONAL];
}

message ValidateReceiptResponse {
  // The updated subscription status.
  SubscriptionStatus status = 1;

  // Whether the receipt was valid.
  bool valid = 2;

  // Error message if validation failed.
  string error_message = 3;
}

message RestorePurchasesRequest {
  // The parent user resource.
  // Format: users/{user}
  string parent = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "memos.api.v1/User"}
  ];
}

message RestorePurchasesResponse {
  // The restored subscription status.
  SubscriptionStatus status = 1;

  // Whether any purchases were restored.
  bool restored = 2;

  // Number of subscriptions restored.
  int32 restored_count = 3;
}

message GetStorageUsageRequest {
  // The resource name of the user.
  // Format: users/{user}
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "memos.api.v1/User"}
  ];
}

message HandleAppleNotificationRequest {
  // The signed notification from Apple (JWS format).
  string signed_payload = 1 [(google.api.field_behavior) = REQUIRED];
}

message ListSubscriptionHistoryRequest {
  // The parent user resource.
  // Format: users/{user}
  string parent = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "memos.api.v1/User"}
  ];

  // Maximum number of results to return.
  int32 page_size = 2 [(google.api.field_behavior) = OPTIONAL];

  // Page token for pagination.
  string page_token = 3 [(google.api.field_behavior) = OPTIONAL];
}

message ListSubscriptionHistoryResponse {
  // List of subscription history events.
  repeated SubscriptionHistoryEvent events = 1;

  // Token for next page.
  string next_page_token = 2;

  // Total count.
  int32 total_size = 3;
}

// SubscriptionHistoryEvent represents a subscription history event.
message SubscriptionHistoryEvent {
  // The event type.
  EventType event_type = 1;

  // The event timestamp.
  google.protobuf.Timestamp event_time = 2;

  // The product ID.
  string product_id = 3;

  // Previous status.
  string from_status = 4;

  // New status.
  string to_status = 5;

  // Additional details.
  map<string, string> details = 6;

  enum EventType {
    EVENT_TYPE_UNSPECIFIED = 0;
    PURCHASE = 1;
    RENEWAL = 2;
    CANCEL = 3;
    EXPIRE = 4;
    REFUND = 5;
    TRIAL_START = 6;
    TRIAL_END = 7;
    GRACE_PERIOD_START = 8;
    GRACE_PERIOD_END = 9;
  }
}
```

### 4.2 Go数据模型

创建新文件：`store/subscription.go`

```go
package store

import (
	"context"
	"time"
)

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusActive        SubscriptionStatus = "ACTIVE"
	SubscriptionStatusExpired       SubscriptionStatus = "EXPIRED"
	SubscriptionStatusCancelled     SubscriptionStatus = "CANCELLED"
	SubscriptionStatusGracePeriod   SubscriptionStatus = "GRACE_PERIOD"
	SubscriptionStatusBillingRetry  SubscriptionStatus = "BILLING_RETRY"
)

// VipType represents the type of VIP.
type VipType string

const (
	VipTypeNone         VipType = "NONE"
	VipTypeTrial        VipType = "TRIAL"
	VipTypeSubscription VipType = "SUBSCRIPTION"
)

// UserSubscription represents a user's subscription.
type UserSubscription struct {
	ID int32

	// User reference
	UserID int32

	// Apple transaction identifiers
	OriginalTransactionID string
	TransactionID         string
	ProductID             string

	// Subscription status
	Status SubscriptionStatus

	// Timestamps
	PurchaseDateTs       int64
	ExpiresDateTs        int64
	CancellationDateTs   *int64
	GracePeriodExpiresTs *int64

	// Trial and intro offer
	IsTrialPeriod    bool
	IsInIntroOffer   bool

	// Environment
	Environment string // "PRODUCTION" or "SANDBOX"

	// Apple notification tracking
	LastNotificationType *string
	LastNotificationTs   *int64

	// Receipt data (encrypted)
	ReceiptData *string

	// Standard fields
	CreatedTs int64
	UpdatedTs int64
}

// UserVIPStatus represents a user's VIP status.
type UserVIPStatus struct {
	ID int32

	// User reference
	UserID int32

	// VIP status
	IsVIP   bool
	VipType VipType

	// Trial information
	TrialStartTs *int64
	TrialEndTs   *int64
	TrialUsed    bool

	// Subscription reference
	SubscriptionID *int32

	// Standard fields
	CreatedTs int64
	UpdatedTs int64
}

// UserStorageUsage represents a user's storage usage.
type UserStorageUsage struct {
	ID int32

	// User reference
	UserID int32

	// Usage in bytes
	TotalBytes         int64
	AttachmentBytes    int64
	MemoContentBytes   int64

	// Quota in bytes
	QuotaBytes int64

	// Last calculation timestamp
	LastCalculatedTs int64

	// Standard fields
	CreatedTs int64
	UpdatedTs int64
}

// SubscriptionHistory represents a subscription history event.
type SubscriptionHistory struct {
	ID int32

	// User reference
	UserID int32

	// Event information
	EventType string
	EventTs   int64

	// Subscription information
	OriginalTransactionID *string
	ProductID             *string

	// Status change
	FromStatus *string
	ToStatus   *string

	// Details (JSON)
	Details *string

	// Standard fields
	CreatedTs int64
}

// Constants for storage quotas
const (
	DefaultQuotaBytes    = 50 * 1024 * 1024  // 50MB
	VIPQuotaBytes       = 5 * 1024 * 1024 * 1024  // 5GB
	TrialDurationDays   = 10
)
```

---

## 5. iOS前端设计

### 5.1 新增文件结构

```
MoeMemos/
├── Packages/
│   ├── Subscription/
│   │   ├── Sources/
│   │   │   ├── Subscription/
│   │   │   │   ├── Models/
│   │   │   │   │   ├── SubscriptionStatus.swift
│   │   │   │   │   ├── Product.swift
│   │   │   │   │   └── StorageUsage.swift
│   │   │   │   ├── Services/
│   │   │   │   │   ├── SubscriptionService.swift
│   │   │   │   │   ├── StoreKitManager.swift
│   │   │   │   │   └── ReceiptValidator.swift
│   │   │   │   ├── ViewModels/
│   │   │   │   │   └── SubscriptionViewModel.swift
│   │   │   │   └── Views/
│   │   │   │       ├── SubscriptionView.swift
│   │   │   │       ├── SubscriptionStatusView.swift
│   │   │   │       ├── PurchaseView.swift
│   │   │   │       └── StorageUsageView.swift
│   │   │   └── ...
│   │   └── Package.swift
│   └── ...
```

### 5.2 StoreKit管理器核心代码

```swift
import Foundation
import StoreKit

@MainActor
public class StoreKitManager: ObservableObject {
    @Published public private(set) var products: [Product] = []
    @Published public private(set) var purchasedSubscriptions: [Product] = []
    @Published public private(set) var isLoading = false
    @Published public var error: Error?
    
    private let productIDs = ["com.memos.vip.yearly"]
    private var updateListenerTask: Task<Void, Error>?
    
    public init() {
        updateListenerTask = listenForTransactions()
    }
    
    deinit {
        updateListenerTask?.cancel()
    }
    
    public func loadProducts() async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            let storeProducts = try await Product.products(for: productIDs)
            products = storeProducts.sorted { $0.price < $1.price }
            await updatePurchasedSubscriptions()
        } catch {
            self.error = error
        }
    }
    
    public func purchase(_ product: Product) async throws -> Transaction? {
        let result = try await product.purchase()
        
        switch result {
        case .success(let verification):
            let transaction = try checkVerified(verification)
            await transaction.finish()
            await updatePurchasedSubscriptions()
            return transaction
            
        case .userCancelled:
            throw StoreKitError.userCancelled
            
        case .pending:
            throw StoreKitError.pending
            
        @unknown default:
            throw StoreKitError.unknown
        }
    }
    
    public func restorePurchases() async throws {
        try await AppStore.sync()
        await updatePurchasedSubscriptions()
    }
    
    private func listenForTransactions() -> Task<Void, Error> {
        return Task.detached {
            for await result in Transaction.updates {
                do {
                    let transaction = try self.checkVerified(result)
                    await transaction.finish()
                    await self.updatePurchasedSubscriptions()
                } catch {
                    print("Transaction verification failed: \(error)")
                }
            }
        }
    }
    
    private func updatePurchasedSubscriptions() async {
        var purchased: [Product] = []
        
        for await result in Transaction.currentEntitlements {
            if case .verified(let transaction) = result {
                if let product = products.first(where: { $0.id == transaction.productID }) {
                    purchased.append(product)
                }
            }
        }
        
        purchasedSubscriptions = purchased
    }
    
    private func checkVerified<T>(_ result: VerificationResult<T>) throws -> T {
        switch result {
        case .verified(let safe):
            return safe
        case .unverified(_, let verificationError):
            throw verificationError
        }
    }
}
```

---

## 6. StoreKit集成

### 6.1 StoreKit配置文件

创建 `MoeMemos/Resources/Subscription.storekit`:

```json
{
  "identifier" : "Subscription",
  "nonRenewingSubscriptions" : [],
  "products" : [],
  "settings" : {
    "_failTransactionsEnabled" : false,
    "_locale" : "en_US",
    "_storefront" : "USA",
    "_storeKitErrors" : []
  },
  "subscriptionGroups" : [
    {
      "id" : "21598467",
      "localizations" : {},
      "name" : "VIP Subscription",
      "subscriptions" : [
        {
          "adHocOffers" : [],
          "codeOffers" : [],
          "displayPrice" : "9.99",
          "familyShareable" : false,
          "groupNumber" : 21598467,
          "internalID" : "6738841625",
          "introductoryOffer" : null,
          "localizations" : [
            {
              "description" : "Unlock unlimited storage and premium features",
              "displayName" : "VIP Annual Subscription",
              "locale" : "en_US"
            },
            {
              "description" : "解锁无限存储空间和高级功能",
              "displayName" : "VIP年度订阅",
              "locale" : "zh_CN"
            }
          ],
          "productID" : "com.memos.vip.yearly",
          "recurringSubscriptionPeriod" : "P1Y",
          "referenceName" : "VIP Annual",
          "subscriptionGroupID" : "21598467",
          "type" : "RecurringSubscription"
        }
      ]
    }
  ],
  "version" : {
    "major" : 4,
    "minor" : 0
  }
}
```

### 6.2 本地测试配置

1. **在Xcode中启用StoreKit配置**:
   - Product → Scheme → Edit Scheme
   - Run → Options → StoreKit Configuration
   - 选择 `Subscription.storekit`

2. **测试购买流程**:
   - 使用模拟器或真机
   - StoreKit会使用本地配置文件，无需真实支付
   - 可以测试各种场景：成功、取消、失败等

3. **测试交易管理**:
   - Transaction Manager: Debug → StoreKit → Manage Transactions
   - 可以查看、删除测试交易
   - 可以模拟退款

---

## 7. Apple服务器通知处理

### 7.1 通知类型

Apple App Store Server Notifications V2 支持以下通知类型：

| 通知类型 | 说明 | 处理方式 |
|---------|------|---------|
| `SUBSCRIBED` | 新订阅 | 创建订阅记录，激活VIP |
| `DID_RENEW` | 续费成功 | 更新到期时间，激活VIP |
| `DID_FAIL_TO_RENEW` | 续费失败 | 进入宽限期 |
| `DID_CHANGE_RENEWAL_STATUS` | 自动续费状态变更 | 更新willRenew标志 |
| `EXPIRED` | 订阅过期 | 取消VIP，降级存储配额 |
| `GRACE_PERIOD_EXPIRED` | 宽限期结束 | 取消VIP |
| `REFUND` | 退款 | 取消VIP，记录退款历史 |
| `REVOKE` | 家庭共享取消 | 取消VIP |

### 7.2 配置App Store Connect

1. **在App Store Connect中配置服务器通知URL**:
   - 登录 App Store Connect
   - 选择你的App
   - App Information → App Store Server Notifications
   - 设置URL: `https://your-domain.com/api/v1/apple/notifications`
   - 选择版本: Version 2

---

## 8. 业务流程

### 8.1 新用户注册流程

```
1. 用户注册账号
   ↓
2. 创建用户记录
   ↓
3. 初始化VIP状态（is_vip=true, vip_type=TRIAL）
   ↓
4. 设置试用期（10天）
   ↓
5. 初始化存储配额（5GB）
   ↓
6. 返回成功响应
```

### 8.2 存储配额检查流程

```
用户上传附件
   ↓
1. 获取用户存储使用量
   ↓
2. 计算新文件大小 + 当前使用量
   ↓
3. 检查是否超过配额
   ├─ 未超过 → 允许上传 → 更新存储使用量
   └─ 已超过 → 拒绝上传 → 返回错误：存储空间不足
```

### 8.3 订阅购买流程

```
iOS客户端
   ↓
1. 用户点击订阅按钮
   ↓
2. StoreKit显示购买界面
   ↓
3. 用户确认购买
   ↓
4. Apple处理支付
   ↓
5. 返回收据给客户端
   ↓
6. 客户端发送收据到后端
   ↓
后端处理
   ↓
7. 验证收据（调用Apple API）
   ↓
8. 创建/更新订阅记录
   ↓
9. 更新VIP状态
   ↓
10. 更新存储配额
   ↓
11. 返回订阅状态给客户端
```

---

## 9. 苹果审核合规

### 9.1 必须满足的要求

#### 9.1.1 应用内购买入口

✅ **要求**: VIP功能必须在应用内提供购买选项

**实现**:
- 在设置页面添加"订阅VIP"按钮
- 点击后显示StoreKit购买界面
- 不能引导用户到网站购买

#### 9.1.2 恢复购买按钮

✅ **要求**: 必须提供恢复购买功能

**实现**:
- 在订阅页面添加"恢复购买"按钮
- 调用 `AppStore.sync()` 恢复购买
- 后端同步订阅状态

#### 9.1.3 订阅条款说明

✅ **要求**: 必须清晰说明订阅条款、价格、续费规则

**实现**:
- 在购买界面显示：
  - 订阅价格
  - 订阅周期
  - 自动续费说明
  - 取消订阅方法
  - 隐私政策链接
  - 服务条款链接

#### 9.1.4 不能绕过IAP

✅ **要求**: VIP功能不能在应用外购买或解锁

**实现**:
- 所有VIP订阅必须通过IAP
- 不能显示外部购买链接
- 不能引导用户到网站支付

### 9.2 审核注意事项

1. **订阅管理页面必须包含**:
   - 当前订阅状态
   - 到期时间
   - 取消订阅说明
   - 恢复购买按钮

2. **价格显示**:
   - 必须显示本地货币价格
   - 必须说明是周期性订阅

3. **自动续费说明**:
   - 必须说明会在到期前24小时扣费
   - 必须说明如何关闭自动续费

4. **试用期说明**:
   - 如果提供试用期，必须说明：
     - 试用期时长
     - 试用期结束后会自动扣费
     - 如何在扣费前取消

---

## 10. 测试方案

### 10.1 StoreKit本地测试

**无需付费开发者账号**，使用Xcode内置的StoreKit测试：

1. **创建StoreKit配置文件**:
   - File → New → File
   - 选择 StoreKit Configuration File
   - 添加订阅产品

2. **启用配置**:
   - Product → Scheme → Edit Scheme
   - Run → Options → StoreKit Configuration
   - 选择创建的配置文件

3. **测试场景**:
   - ✅ 成功购买
   - ✅ 用户取消
   - ✅ 网络错误
   - ✅ 购买失败
   - ✅ 恢复购买
   - ✅ 订阅续费
   - ✅ 订阅过期
   - ✅ 退款

4. **Transaction Manager**:
   - Debug → StoreKit → Manage Transactions
   - 查看所有测试交易
   - 删除交易重新测试
   - 模拟退款

### 10.2 Sandbox测试

**需要付费开发者账号**：

1. **创建Sandbox测试账号**:
   - App Store Connect → Users and Access
   - Sandbox Testers
   - 创建测试账号

2. **在设备上登录**:
   - Settings → App Store → Sandbox Account
   - 登录测试账号

3. **测试流程**:
   - 使用测试账号购买
   - 不会真实扣费
   - 可以测试各种订阅状态

### 10.3 后端测试

1. **收据验证测试**:
   - 使用Sandbox收据
   - 调用Apple Sandbox API验证
   - 测试各种收据状态

2. **通知处理测试**:
   - 使用Apple提供的测试通知
   - 验证签名解析
   - 测试各种通知类型

---

## 11. 部署计划

### 11.1 开发阶段

1. **后端开发** (2-3周):
   - [ ] 数据库迁移
   - [ ] Store层实现
   - [ ] API服务实现
   - [ ] 收据验证
   - [ ] 通知处理
   - [ ] 存储配额管理
   - [ ] 试用期管理

2. **iOS前端开发** (2-3周):
   - [ ] StoreKit集成
   - [ ] 订阅UI界面
   - [ ] 订阅状态管理
   - [ ] 存储配额显示
   - [ ] 本地化

3. **测试阶段** (1-2周):
   - [ ] 单元测试
   - [ ] 集成测试
   - [ ] StoreKit本地测试
   - [ ] Sandbox测试

### 11.2 上线准备

1. **App Store Connect配置**:
   - [ ] 创建App ID
   - [ ] 创建订阅组
   - [ ] 创建订阅产品
   - [ ] 配置服务器通知URL
   - [ ] 上传截图和描述

2. **后端部署**:
   - [ ] 部署新版本
   - [ ] 运行数据库迁移
   - [ ] 配置HTTPS证书
   - [ ] 配置Apple通知Webhook

3. **提交审核**:
   - [ ] 准备审核材料
   - [ ] 提交App审核
   - [ ] 准备回复审核问题

### 11.3 上线后监控

1. **监控指标**:
   - 订阅购买转化率
   - 订阅续费率
   - 订阅取消率
   - 退款率
   - 存储配额使用情况

2. **告警设置**:
   - Apple通知处理失败
   - 收据验证失败
   - 订阅状态异常

---

## 附录

### A. 相关文档

- [Apple In-App Purchase](https://developer.apple.com/in-app-purchase/)
- [StoreKit 2 Documentation](https://developer.apple.com/documentation/storekit)
- [App Store Server Notifications V2](https://developer.apple.com/documentation/appstoreservernotifications)
- [App Store Review Guidelines](https://developer.apple.com/app-store/review/guidelines/#subscriptions)

### B. 价格建议

建议定价：
- 年度订阅：¥68/年 或 $9.99/年
- 可根据市场反馈调整

### C. 未来扩展

1. **多订阅等级**:
   - 基础版：50MB → 500MB
   - 专业版：50MB → 5GB
   - 企业版：50MB → 50GB

2. **家庭共享**:
   - 支持家庭共享订阅
   - 一个订阅多人使用

3. **优惠活动**:
   - 新用户首年折扣
   - 节日促销
   - 推荐奖励

---

**文档结束**

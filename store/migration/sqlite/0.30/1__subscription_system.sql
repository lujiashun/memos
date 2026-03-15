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

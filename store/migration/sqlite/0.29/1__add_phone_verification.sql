-- Add phone_number column to user table
ALTER TABLE user ADD COLUMN phone_number TEXT NOT NULL DEFAULT '';

-- Create verification table
CREATE TABLE verification (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  phone_number TEXT NOT NULL,
  method TEXT NOT NULL CHECK (method IN ('SMS', 'PHONE_AUTH')),
  purpose TEXT NOT NULL CHECK (purpose IN ('REGISTER', 'FORGOT_PASSWORD')),
  code TEXT NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  expires_ts BIGINT NOT NULL,
  is_used INTEGER NOT NULL CHECK (is_used IN (0, 1)) DEFAULT 0,
  UNIQUE(phone_number, purpose, created_ts)
);

-- Create indexes
CREATE INDEX idx_verification_phone_purpose ON verification(phone_number, purpose);
CREATE INDEX idx_verification_expires ON verification(expires_ts);
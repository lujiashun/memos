-- Add phone_number column to user table
ALTER TABLE "user" ADD COLUMN phone_number VARCHAR(20) NOT NULL DEFAULT '';

-- Create verification table
CREATE TABLE verification (
  id SERIAL PRIMARY KEY,
  phone_number VARCHAR(20) NOT NULL,
  method VARCHAR(20) NOT NULL CHECK (method IN ('SMS', 'PHONE_AUTH')),
  purpose VARCHAR(20) NOT NULL CHECK (purpose IN ('REGISTER', 'FORGOT_PASSWORD')),
  code VARCHAR(50) NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  expires_ts BIGINT NOT NULL,
  is_used BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE (phone_number, purpose, created_ts)
);

-- Create indexes
CREATE INDEX idx_verification_phone_purpose ON verification(phone_number, purpose);
CREATE INDEX idx_verification_expires ON verification(expires_ts);
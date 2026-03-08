-- Add phone_number column to user table
ALTER TABLE user ADD COLUMN phone_number VARCHAR(20) NOT NULL DEFAULT '';

-- Create verification table
CREATE TABLE verification (
  id INT PRIMARY KEY AUTO_INCREMENT,
  phone_number VARCHAR(20) NOT NULL,
  method ENUM('SMS', 'PHONE_AUTH') NOT NULL,
  purpose ENUM('REGISTER', 'FORGOT_PASSWORD') NOT NULL,
  code VARCHAR(50) NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT UNIX_TIMESTAMP(),
  expires_ts BIGINT NOT NULL,
  is_used TINYINT NOT NULL DEFAULT 0 CHECK (is_used IN (0, 1)),
  UNIQUE KEY (phone_number, purpose, created_ts)
);

-- Create indexes
CREATE INDEX idx_verification_phone_purpose ON verification(phone_number, purpose);
CREATE INDEX idx_verification_expires ON verification(expires_ts);
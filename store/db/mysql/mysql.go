package mysql

import (
	"context"
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
)

type DB struct {
	db      *sql.DB
	profile *profile.Profile
	config  *mysql.Config
}

func NewDB(profile *profile.Profile) (store.Driver, error) {
	// Open MySQL connection with parameter.
	// multiStatements=true is required for migration.
	// See more in: https://github.com/go-sql-driver/mysql#multistatements
	dsn, err := mergeDSN(profile.DSN)
	if err != nil {
		return nil, err
	}

	driver := DB{profile: profile}
	driver.config, err = mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("Parse DSN eroor")
	}

	driver.db, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db: %s", profile.DSN)
	}

	return &driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'memo' AND TABLE_TYPE = 'BASE TABLE')").Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if database is initialized")
	}
	return exists, nil
}

func mergeDSN(baseDSN string) (string, error) {
	config, err := mysql.ParseDSN(baseDSN)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse DSN: %s", baseDSN)
	}

	config.MultiStatements = true
	return config.FormatDSN(), nil
}

// CreateVerification 创建验证记录
func (d *DB) CreateVerification(ctx context.Context, create *store.Verification) (*store.Verification, error) {
	query := `
		INSERT INTO verification (
			phone_number, method, purpose, code, created_ts, expires_ts, is_used
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := d.db.ExecContext(
		ctx,
		query,
		create.PhoneNumber,
		create.Method,
		create.Purpose,
		create.Code,
		create.CreatedTs,
		create.ExpiresTs,
		create.IsUsed,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create verification")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get last insert id")
	}
	create.ID = int32(id)
	return create, nil
}

// GetVerification 获取验证记录
func (d *DB) GetVerification(ctx context.Context, phoneNumber, code, purpose string) (*store.Verification, error) {
	query := `
		SELECT id, phone_number, method, purpose, code, created_ts, expires_ts, is_used
		FROM verification
		WHERE phone_number = ? AND code = ? AND purpose = ? AND expires_ts > UNIX_TIMESTAMP() AND is_used = 0
		ORDER BY created_ts DESC
		LIMIT 1
	`
	verification := &store.Verification{}
	err := d.db.QueryRowContext(
		ctx,
		query,
		phoneNumber,
		code,
		purpose,
	).Scan(
		&verification.ID,
		&verification.PhoneNumber,
		&verification.Method,
		&verification.Purpose,
		&verification.Code,
		&verification.CreatedTs,
		&verification.ExpiresTs,
		&verification.IsUsed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get verification")
	}
	return verification, nil
}

// UpdateVerification 更新验证记录
func (d *DB) UpdateVerification(ctx context.Context, update *store.Verification) error {
	query := `
		UPDATE verification
		SET is_used = ?
		WHERE id = ?
	`
	_, err := d.db.ExecContext(
		ctx,
		query,
		update.IsUsed,
		update.ID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update verification")
	}
	return nil
}

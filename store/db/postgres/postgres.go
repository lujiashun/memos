package postgres

import (
	"context"
	"database/sql"
	"log"

	// Import the PostgreSQL driver.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
)

type DB struct {
	db      *sql.DB
	profile *profile.Profile
}

func NewDB(profile *profile.Profile) (store.Driver, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}

	// Open the PostgreSQL connection
	db, err := sql.Open("postgres", profile.DSN)
	if err != nil {
		log.Printf("Failed to open database: %s", err)
		return nil, errors.Wrapf(err, "failed to open database: %s", profile.DSN)
	}

	var driver store.Driver = &DB{
		db:      db,
		profile: profile,
	}

	// Return the DB struct
	return driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_catalog = current_database() AND table_name = 'memo' AND table_type = 'BASE TABLE')").Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if database is initialized")
	}
	return exists, nil
}

// CreateVerification 创建验证记录
func (d *DB) CreateVerification(ctx context.Context, create *store.Verification) (*store.Verification, error) {
	query := `
		INSERT INTO verification (
			phone_number, method, purpose, code, created_ts, expires_ts, is_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id int32
	err := d.db.QueryRowContext(
		ctx,
		query,
		create.PhoneNumber,
		create.Method,
		create.Purpose,
		create.Code,
		create.CreatedTs,
		create.ExpiresTs,
		create.IsUsed,
	).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create verification")
	}
	create.ID = id
	return create, nil
}

// GetVerification 获取验证记录
func (d *DB) GetVerification(ctx context.Context, phoneNumber, code, purpose string) (*store.Verification, error) {
	query := `
		SELECT id, phone_number, method, purpose, code, created_ts, expires_ts, is_used
		FROM verification
		WHERE phone_number = $1 AND code = $2 AND purpose = $3 AND expires_ts > EXTRACT(EPOCH FROM NOW()) AND is_used = false
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
		SET is_used = $1
		WHERE id = $2
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

package sqlite

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	// Note: modernc.org/sqlite driver is imported in functions.go where
	// RegisterScalarFunction is used. No blank import needed here.

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
)

type DB struct {
	db      *sql.DB
	profile *profile.Profile
}

// NewDB opens a database specified by its database driver name and a
// driver-specific data source name, usually consisting of at least a
// database name and connection information.
func NewDB(profile *profile.Profile) (store.Driver, error) {
	// Ensure a DSN is set before attempting to open the database.
	if profile.DSN == "" {
		return nil, errors.New("dsn required")
	}

	if err := ensureUnicodeLowerRegistered(); err != nil {
		return nil, errors.Wrap(err, "failed to register sqlite unicode lower function")
	}

	// Connect to the database with some sane settings:
	// - No shared-cache: it's obsolete; WAL journal mode is a better solution.
	// - No foreign key constraints: it's currently disabled by default, but it's a
	// good practice to be explicit and prevent future surprises on SQLite upgrades.
	// - Journal mode set to WAL: it's the recommended journal mode for most applications
	// as it prevents locking issues.
	// - mmap size set to 0: it disables memory mapping, which can cause OOM errors on some systems.
	//
	// Notes:
	// - When using the `modernc.org/sqlite` driver, each pragma must be prefixed with `_pragma=`.
	//
	// References:
	// - https://pkg.go.dev/modernc.org/sqlite#Driver.Open
	// - https://www.sqlite.org/sharedcache.html
	// - https://www.sqlite.org/pragma.html
	sqliteDB, err := sql.Open("sqlite", profile.DSN+"?_pragma=foreign_keys(0)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=mmap_size(0)")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db with dsn: %s", profile.DSN)
	}

	driver := DB{db: sqliteDB, profile: profile}

	return &driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	// Check if the database is initialized by checking if the memo table exists.
	var exists bool
	err := d.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='memo')").Scan(&exists)
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
		) VALUES (?, ?, ?, ?, ?, ?, ?)
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
		WHERE phone_number = ? AND code = ? AND purpose = ? AND expires_ts > strftime('%s', 'now') AND is_used = 0
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

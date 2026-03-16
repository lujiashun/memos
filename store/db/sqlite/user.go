package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) CreateUser(ctx context.Context, create *store.User) (*store.User, error) {
	fields := []string{"`username`", "`role`", "`email`", "`nickname`", "`password_hash`", "`avatar_url`", "`phone_number`"}
	placeholder := []string{"?", "?", "?", "?", "?", "?", "?"}
	args := []any{create.Username, create.Role, create.Email, create.Nickname, create.PasswordHash, create.AvatarURL, create.PhoneNumber}
	stmt := "INSERT INTO user (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(placeholder, ", ") + ") RETURNING id, description, created_ts, updated_ts, row_status"
	if err := d.db.QueryRowContext(ctx, stmt, args...).Scan(
		&create.ID,
		&create.Description,
		&create.CreatedTs,
		&create.UpdatedTs,
		&create.RowStatus,
	); err != nil {
		return nil, err
	}

	return create, nil
}

func (d *DB) UpdateUser(ctx context.Context, update *store.UpdateUser) (*store.User, error) {
	set, args := []string{}, []any{}
	if v := update.UpdatedTs; v != nil {
		set, args = append(set, "updated_ts = ?"), append(args, *v)
	}
	if v := update.RowStatus; v != nil {
		set, args = append(set, "row_status = ?"), append(args, *v)
	}
	if v := update.Username; v != nil {
		set, args = append(set, "username = ?"), append(args, *v)
	}
	if v := update.Email; v != nil {
		set, args = append(set, "email = ?"), append(args, *v)
	}
	if v := update.Nickname; v != nil {
		set, args = append(set, "nickname = ?"), append(args, *v)
	}
	if v := update.AvatarURL; v != nil {
		set, args = append(set, "avatar_url = ?"), append(args, *v)
	}
	if v := update.PasswordHash; v != nil {
		set, args = append(set, "password_hash = ?"), append(args, *v)
	}
	if v := update.Description; v != nil {
		set, args = append(set, "description = ?"), append(args, *v)
	}
	if v := update.Role; v != nil {
		set, args = append(set, "role = ?"), append(args, *v)
	}
	if v := update.PhoneNumber; v != nil {
		set, args = append(set, "phone_number = ?"), append(args, *v)
	}
	args = append(args, update.ID)

	query := `
		UPDATE user
		SET ` + strings.Join(set, ", ") + `
		WHERE id = ?
		RETURNING id, username, role, email, nickname, password_hash, avatar_url, description, phone_number, created_ts, updated_ts, row_status
	`
	user := &store.User{}
	if err := d.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.Role,
		&user.Email,
		&user.Nickname,
		&user.PasswordHash,
		&user.AvatarURL,
		&user.Description,
		&user.PhoneNumber,
		&user.CreatedTs,
		&user.UpdatedTs,
		&user.RowStatus,
	); err != nil {
		return nil, err
	}

	return user, nil
}

func (d *DB) ListUsers(ctx context.Context, find *store.FindUser) ([]*store.User, error) {
	where, args := []string{"1 = 1"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.Username; v != nil {
		// Use exact match for username to avoid matching similar usernames (e.g., "lujs" matching "lujs456")
		where, args = append(where, "username = ?"), append(args, *v)
	}
	if v := find.Role; v != nil {
		where, args = append(where, "role = ?"), append(args, *v)
	}
	if v := find.Email; v != nil {
		where, args = append(where, "email = ?"), append(args, *v)
	}
	if v := find.Nickname; v != nil {
		where, args = append(where, "nickname = ?"), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
	}

	// Handle search filter (searches across username, email, nickname)
	if len(find.Filters) > 0 && find.Filters[0] != "" {
		searchTerm := "%" + find.Filters[0] + "%"
		where = append(where, "(username LIKE ? OR email LIKE ? OR nickname LIKE ?)")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Determine order by clause
	orderBy := []string{"created_ts DESC", "id DESC"}
	if find.OrderBy != nil && *find.OrderBy != "" {
		orderBy = []string{*find.OrderBy}
	}

	// Cursor-based pagination: add cursor condition
	if find.CursorCreatedTs != nil && find.CursorID != nil {
		// For DESC order on created_ts
		if len(orderBy) > 0 && strings.Contains(orderBy[0], "created_ts DESC") {
			where = append(where, "(created_ts < ? OR (created_ts = ? AND id < ?))")
			args = append(args, *find.CursorCreatedTs, *find.CursorCreatedTs, *find.CursorID)
		} else if len(orderBy) > 0 && strings.Contains(orderBy[0], "created_ts ASC") {
			// For ASC order on created_ts
			where = append(where, "(created_ts > ? OR (created_ts = ? AND id > ?))")
			args = append(args, *find.CursorCreatedTs, *find.CursorCreatedTs, *find.CursorID)
		}
	}

	query := `
		SELECT 
			id,
			username,
			role,
			email,
			nickname,
			password_hash,
			avatar_url,
			description,
			phone_number,
			created_ts,
			updated_ts,
			row_status
		FROM user
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + strings.Join(orderBy, ", ")
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*store.User, 0)
	for rows.Next() {
		var user store.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Role,
			&user.Email,
			&user.Nickname,
			&user.PasswordHash,
			&user.AvatarURL,
			&user.Description,
			&user.PhoneNumber,
			&user.CreatedTs,
			&user.UpdatedTs,
			&user.RowStatus,
		); err != nil {
			return nil, err
		}
		list = append(list, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DB) DeleteUser(ctx context.Context, delete *store.DeleteUser) error {
	result, err := d.db.ExecContext(ctx, `
		DELETE FROM user WHERE id = ?
	`, delete.ID)
	if err != nil {
		return err
	}
	if _, err := result.RowsAffected(); err != nil {
		return err
	}
	return nil
}

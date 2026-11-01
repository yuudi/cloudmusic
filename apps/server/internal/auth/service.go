package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrInvalid = errors.New("invalid input")
var ErrUnauthorized = errors.New("unauthorized")
var ErrConflict = errors.New("username already in use")
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

const lifetime = 30 * 24 * time.Hour

type User struct {
	ID          string  `json:"id"`
	Username    *string `json:"username"`
	DisplayName string  `json:"display_name"`
	IsGuest     bool    `json:"is_guest"`
}
type Service struct{ DB *sql.DB }

func random(n int) ([]byte, error) { b := make([]byte, n); _, err := rand.Read(b); return b, err }
func newID() (string, error) {
	b, e := random(16)
	if e != nil {
		return "", e
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	s := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", s[:8], s[8:12], s[12:16], s[16:20], s[20:]), nil
}
func validName(s string) bool { return usernamePattern.MatchString(s) }
func validDisplay(s string) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= 100 && !strings.ContainsRune(s, 0)
}
func conflict(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		return ErrConflict
	}
	return err
}
func tokenHash(token string) []byte { h := sha256.Sum256([]byte(token)); return h[:] }
func createSession(ctx context.Context, tx *sql.Tx, id string) (string, error) {
	b, e := random(32)
	if e != nil {
		return "", e
	}
	t := base64.RawURLEncoding.EncodeToString(b)
	_, e = tx.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES ($1,$2,now()+interval '30 days')`, tokenHash(t), id)
	return t, e
}
func (s *Service) Create(ctx context.Context, username, password, display string, guest bool) (User, string, error) {
	var u User
	if !validDisplay(display) || (!guest && !validName(username)) {
		return u, "", ErrInvalid
	}
	var hash string
	var e error
	if !guest {
		hash, e = PasswordHash(password)
		if e != nil {
			return u, "", e
		}
	}
	u.ID, e = newID()
	if e != nil {
		return u, "", e
	}
	u.IsGuest = guest
	u.DisplayName = display
	if !guest {
		u.Username = &username
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return u, "", e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `INSERT INTO users(id,username,display_name,is_guest) VALUES($1,$2,$3,$4)`, u.ID, u.Username, display, guest)
	if e != nil {
		return u, "", conflict(e)
	}
	if !guest {
		if _, e = tx.ExecContext(ctx, `INSERT INTO password_credentials VALUES($1,$2)`, u.ID, hash); e != nil {
			return u, "", e
		}
	}
	token, e := createSession(ctx, tx, u.ID)
	if e != nil {
		return u, "", e
	}
	return u, token, tx.Commit()
}
func (s *Service) Login(ctx context.Context, username, password string) (User, string, error) {
	var u User
	var hash string
	if !validName(username) || len(password) > 1024 {
		return u, "", ErrUnauthorized
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return u, "", e
	}
	defer tx.Rollback()
	e = tx.QueryRowContext(ctx, `SELECT id,username,display_name,is_guest FROM users WHERE lower(username)=lower($1) AND deleted_at IS NULL FOR UPDATE`, username).Scan(&u.ID, &u.Username, &u.DisplayName, &u.IsGuest)
	if errors.Is(e, sql.ErrNoRows) {
		return u, "", ErrUnauthorized
	}
	if e != nil {
		return u, "", e
	}
	// A separate statement after acquiring the user lock obtains a fresh snapshot.
	e = tx.QueryRowContext(ctx, `SELECT password_hash FROM password_credentials WHERE user_id=$1`, u.ID).Scan(&hash)
	if errors.Is(e, sql.ErrNoRows) {
		return u, "", ErrUnauthorized
	}
	if e != nil {
		return u, "", e
	}
	valid, rehash := verifyPassword(password, hash)
	if !valid {
		return u, "", ErrUnauthorized
	}
	if rehash {
		upgraded, err := PasswordHash(password)
		if err != nil {
			return u, "", err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE password_credentials SET password_hash=$2 WHERE user_id=$1`, u.ID, upgraded); err != nil {
			return u, "", err
		}
	}
	token, e := createSession(ctx, tx, u.ID)
	if e != nil {
		return u, "", e
	}
	return u, token, tx.Commit()
}

// Authenticate serializes activity, expiry and guest upgrades on the user row.
func (s *Service) Authenticate(ctx context.Context, token string, active bool) (User, error) {
	var u User
	if len(token) != 43 {
		return u, ErrUnauthorized
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return u, e
	}
	defer tx.Rollback()
	e = tx.QueryRowContext(ctx, `SELECT u.id,u.username,u.display_name,u.is_guest FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.deleted_at IS NULL AND (NOT u.is_guest OR u.last_active_at>now()-interval '30 days') FOR UPDATE OF u`, tokenHash(token)).Scan(&u.ID, &u.Username, &u.DisplayName, &u.IsGuest)
	if errors.Is(e, sql.ErrNoRows) {
		return u, ErrUnauthorized
	}
	if e != nil {
		return u, e
	}
	query := `UPDATE sessions SET expires_at=now()+interval '30 days' WHERE token_hash=$1 AND expires_at>now()`
	if !active {
		query = `UPDATE sessions SET expires_at=expires_at WHERE token_hash=$1 AND expires_at>now()`
	}
	res, e := tx.ExecContext(ctx, query, tokenHash(token))
	if e != nil {
		return u, e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return u, e
	}
	if n == 0 {
		return u, ErrUnauthorized
	}
	if active {
		_, e = tx.ExecContext(ctx, `UPDATE users SET last_active_at=now() WHERE id=$1`, u.ID)
		if e != nil {
			return u, e
		}
	}
	return u, tx.Commit()
}
func (s *Service) Upgrade(ctx context.Context, id, username, password string) error {
	if !validName(username) {
		return ErrInvalid
	}
	hash, e := PasswordHash(password)
	if e != nil {
		return e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, `UPDATE users SET username=$2,is_guest=false WHERE id=$1 AND is_guest AND deleted_at IS NULL AND last_active_at>now()-interval '30 days'`, id, username)
	if e != nil {
		return conflict(e)
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return ErrUnauthorized
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO password_credentials VALUES($1,$2)`, id, hash); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Service) Profile(ctx context.Context, id string, username, display *string) error {
	if (display != nil && !validDisplay(*display)) || (username != nil && *username != "" && !validName(*username)) {
		return ErrInvalid
	}
	res, e := s.DB.ExecContext(ctx, `UPDATE users SET display_name=COALESCE($2,display_name),username=COALESCE(NULLIF($3,''),username) WHERE id=$1 AND deleted_at IS NULL AND (COALESCE($3,'')='' OR NOT is_guest)`, id, display, username)
	if e != nil {
		return conflict(e)
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return ErrInvalid
	}
	return nil
}
func (s *Service) Logout(ctx context.Context, id, token string, all bool) error {
	if all {
		_, e := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, id)
		return e
	}
	_, e := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1 AND token_hash=$2`, id, tokenHash(token))
	return e
}
func (s *Service) ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error {
	hash, e := PasswordHash(newPassword)
	if e != nil {
		return e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var lockedID string
	e = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&lockedID)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if e != nil {
		return e
	}
	var old string
	e = tx.QueryRowContext(ctx, `SELECT password_hash FROM password_credentials WHERE user_id=$1`, lockedID).Scan(&old)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if e != nil {
		return e
	}
	if valid, _ := verifyPassword(oldPassword, old); !valid {
		return ErrUnauthorized
	}
	if _, e = tx.ExecContext(ctx, `UPDATE password_credentials SET password_hash=$2 WHERE user_id=$1`, id, hash); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Service) ResetPassword(ctx context.Context, username, password string) error {
	if !validName(username) {
		return ErrInvalid
	}
	hash, e := PasswordHash(password)
	if e != nil {
		return e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var id string
	e = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(username)=lower($1) AND deleted_at IS NULL FOR UPDATE`, username).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE password_credentials SET password_hash=$2 WHERE user_id=$1`, id, hash); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Service) Cleanup(ctx context.Context) error {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback() // Lock rows before revoking sessions; activity and upgrades take the same lock.
	rows, e := tx.QueryContext(ctx, `SELECT id FROM users WHERE is_guest AND deleted_at IS NULL AND last_active_at<=now()-interval '30 days' FOR UPDATE SKIP LOCKED`)
	if e != nil {
		return e
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if _, e = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE users SET display_name='',deleted_at=now() WHERE id=$1 AND is_guest AND deleted_at IS NULL AND last_active_at<=now()-interval '30 days'`, id); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at<=now()`); e != nil {
		return e
	}
	return tx.Commit()
}

package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/yuudi/cloudmusic/apps/server/migrations"
)

func TestPasswordAndValidation(t *testing.T) {
	h, e := PasswordHash("a strong password")
	if e != nil {
		t.Fatal(e)
	}
	if !verify("a strong password", h) || verify("wrong password", h) || verify("password", "bad") {
		t.Fatal("password verification failed")
	}
	if _, e = PasswordHash("short"); !errors.Is(e, ErrInvalid) {
		t.Fatal(e)
	}
	for _, s := range []string{"ab", "bad-name", "中文名"} {
		if validName(s) {
			t.Fatalf("accepted %q", s)
		}
	}
}
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	cfg, e := pgx.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { admin.Close() })
	id := uuid.NewV4().String()
	schema := "test_" + strings.ReplaceAll(id, "-", "")
	if _, e = admin.Exec(`CREATE SCHEMA ` + schema); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if _, e := admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); e != nil {
			t.Error(e)
		}
	})
	cfg.RuntimeParams["search_path"] = schema
	cfg.RuntimeParams["application_name"] = schema
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { db.Close() })
	provider, e := goose.NewProvider(goose.DialectPostgres, db, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = provider.Up(context.Background()); e != nil {
		t.Fatal(e)
	}

	return db
}
func TestAccountLifecycle(t *testing.T) {
	db := testDB(t)
	s := Service{DB: db}
	ctx := context.Background()
	guest, token, e := s.Create(ctx, "", "", "Guest", true)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Upgrade(ctx, guest.ID, "Example_User", "correct horse"); e != nil {
		t.Fatal(e)
	}
	got, e := s.Authenticate(ctx, token, true)
	if e != nil || got.ID != guest.ID || got.IsGuest {
		t.Fatalf("upgrade: %+v %v", got, e)
	}
	if _, _, e = s.Create(ctx, "example_user", "another password", "", false); !errors.Is(e, ErrConflict) {
		t.Fatalf("duplicate: %v", e)
	}
	_, token2, e := s.Login(ctx, "EXAMPLE_USER", "correct horse")
	if e != nil {
		t.Fatal(e)
	}
	username, display := "renamed", "中文"
	if e = s.Profile(ctx, guest.ID, &username, &display); e != nil {
		t.Fatal(e)
	}
	if e = s.ChangePassword(ctx, guest.ID, "correct horse", "new password"); e != nil {
		t.Fatal(e)
	}
	for _, tok := range []string{token, token2} {
		if _, e = s.Authenticate(ctx, tok, true); !errors.Is(e, ErrUnauthorized) {
			t.Fatalf("old session: %v", e)
		}
	}
	_, token, e = s.Login(ctx, "RENAMED", "new password")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ResetPassword(ctx, "renamed", "reset password"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Authenticate(ctx, token, true); !errors.Is(e, ErrUnauthorized) {
		t.Fatalf("reset session: %v", e)
	}
	if _, _, e = s.Login(ctx, "renamed", "reset password"); e != nil {
		t.Fatal(e)
	}
}
func TestGuestExpiry(t *testing.T) {
	db := testDB(t)
	s := Service{DB: db}
	ctx := context.Background()
	guest, token, e := s.Create(ctx, "", "", "Guest", true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`UPDATE users SET last_active_at=now()-interval '31 days' WHERE id=$1`, guest.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Authenticate(ctx, token, true); !errors.Is(e, ErrUnauthorized) {
		t.Fatalf("expired: %v", e)
	}
	if e = s.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	var deleted sql.NullTime
	var display string
	if e = db.QueryRow(`SELECT deleted_at,display_name FROM users WHERE id=$1`, guest.ID).Scan(&deleted, &display); e != nil || !deleted.Valid || display != "" {
		t.Fatalf("cleanup: %v %v %q", e, deleted, display)
	}
	var n int
	if e = db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id=$1`, guest.ID).Scan(&n); e != nil || n != 0 {
		t.Fatalf("sessions: %d %v", n, e)
	}
}
func TestCleanupSkipsLockedUpgrade(t *testing.T) {
	db := testDB(t)
	s := Service{DB: db}
	ctx := context.Background()
	guest, _, e := s.Create(ctx, "", "", "", true)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE users SET last_active_at=now()-interval '31 days' WHERE id=$1`, guest.ID); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- s.Cleanup(ctx) }()
	select {
	case e = <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup blocked on locked user")
	}
	if _, e = tx.Exec(`UPDATE users SET last_active_at=now() WHERE id=$1`, guest.ID); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = s.Upgrade(ctx, guest.ID, "new_user", "long password"); e != nil {
		t.Fatal(e)
	}
	if e = s.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	var active bool
	if e = db.QueryRow(`SELECT deleted_at IS NULL AND NOT is_guest FROM users WHERE id=$1`, guest.ID).Scan(&active); e != nil || !active {
		t.Fatal(fmt.Sprint(active, e))
	}
}

func TestHTTPGuestUpgrade(t *testing.T) {
	db := testDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), &Service{DB: db})
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	guest := request("POST", "/api/v1/auth/guest", "", "")
	if guest.Code != 200 {
		t.Fatalf("guest %d %s", guest.Code, guest.Body.String())
	}
	var result struct {
		Token string `json:"token"`
		User  User   `json:"user"`
	}
	if e := json.Unmarshal(guest.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	// Polling identity must not keep a guest alive; editing their profile must.
	if _, err := db.Exec(`UPDATE users SET last_active_at=now()-interval '1 day' WHERE id=$1`, result.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE sessions SET expires_at=now()+interval '1 hour' WHERE user_id=$1`, result.User.ID); err != nil {
		t.Fatal(err)
	}
	var beforeActivity, beforeExpiry time.Time
	readActivity := func() (activity, expiry time.Time) {
		t.Helper()
		if err := db.QueryRow(`SELECT u.last_active_at,s.expires_at FROM users u JOIN sessions s ON s.user_id=u.id WHERE u.id=$1`, result.User.ID).Scan(&activity, &expiry); err != nil {
			t.Fatal(err)
		}
		return
	}
	beforeActivity, beforeExpiry = readActivity()
	if w := request("GET", "/api/v1/auth/me", "", result.Token); w.Code != 200 {
		t.Fatalf("guest identity check: %d", w.Code)
	}
	if activity, expiry := readActivity(); !activity.Equal(beforeActivity) || !expiry.Equal(beforeExpiry) {
		t.Fatal("identity check changed activity or expiry")
	}
	if w := request("PATCH", "/api/v1/auth/me", `{"display_name":"guest name"}`, result.Token); w.Code != 204 {
		t.Fatalf("guest profile update: %d", w.Code)
	}
	if activity, expiry := readActivity(); !activity.After(beforeActivity) || !expiry.After(beforeExpiry) {
		t.Fatal("profile edit did not refresh activity and expiry")
	}
	upgrade := request("POST", "/api/v1/auth/upgrade", `{"username":"Person","password":"good password"}`, result.Token)
	if upgrade.Code != 204 {
		t.Fatalf("upgrade %d %s", upgrade.Code, upgrade.Body.String())
	}
	me := request("GET", "/api/v1/auth/me", "", result.Token)
	if me.Code != 200 {
		t.Fatalf("me %d", me.Code)
	}
	var current struct {
		User User `json:"user"`
	}
	if e := json.Unmarshal(me.Body.Bytes(), &current); e != nil {
		t.Fatal(e)
	}
	if current.User.ID != result.User.ID || current.User.IsGuest {
		t.Fatalf("identity changed: %+v", current.User)
	}
	if w := request("POST", "/api/v1/auth/logout-all", "", result.Token); w.Code != 204 {
		t.Fatalf("logout %d", w.Code)
	}
	if w := request("GET", "/api/v1/auth/me", "", result.Token); w.Code != 401 {
		t.Fatalf("revoked %d", w.Code)
	}
}

func TestLegacyPasswordUpgradedOnLogin(t *testing.T) {
	db := testDB(t)
	service := Service{DB: db}
	ctx := context.Background()
	user, _, err := service.Create(ctx, "legacy_user", "correct password", "", false)
	if err != nil {
		t.Fatal(err)
	}
	salt := []byte("0123456789abcdef")
	sum := argon2.IDKey([]byte("correct password"), salt, 3, 64*1024, 1, 32)
	legacy := base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(sum)
	if _, err = db.ExecContext(ctx, `UPDATE password_credentials SET password_hash=$2 WHERE user_id=$1`, user.ID, legacy); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.Login(ctx, "legacy_user", "wrong password"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("incorrect password: %v", err)
	}
	var stored string
	if err = db.QueryRowContext(ctx, `SELECT password_hash FROM password_credentials WHERE user_id=$1`, user.ID).Scan(&stored); err != nil || stored != legacy {
		t.Fatalf("invalid login changed hash: %v", err)
	}
	if _, _, err = service.Login(ctx, "legacy_user", "correct password"); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRowContext(ctx, `SELECT password_hash FROM password_credentials WHERE user_id=$1`, user.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$argon2id$v=19$m=65536,t=3,p=4$") || !verify("correct password", stored) {
		t.Fatal("legacy hash was not upgraded")
	}
	if _, _, err = service.Login(ctx, "legacy_user", "correct password"); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordResetSerializesWithLoginAndChange(t *testing.T) {
	for _, kind := range []string{"login", "change"} {
		t.Run(kind, func(t *testing.T) {
			db := testDB(t)
			s := Service{DB: db}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			user, _, err := s.Create(ctx, "race_user", "old password", "", false)
			if err != nil {
				t.Fatal(err)
			}
			replacement, err := PasswordHash("reset password")
			if err != nil {
				t.Fatal(err)
			}
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err = tx.ExecContext(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, user.ID); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.ExecContext(ctx, `UPDATE password_credentials SET password_hash=$2 WHERE user_id=$1`, user.ID, replacement); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				if kind == "login" {
					_, _, e := s.Login(ctx, "race_user", "old password")
					done <- e
				} else {
					done <- s.ChangePassword(ctx, user.ID, "old password", "other password")
				}
			}()
			// Wait until the concurrent operation has started its locking query.
			var appName string
			if err = db.QueryRowContext(ctx, `SELECT current_setting('application_name')`).Scan(&appName); err != nil {
				t.Fatal(err)
			}
			for {
				var waiting bool
				err = db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND query LIKE '%FOR UPDATE%')`, appName).Scan(&waiting)
				if err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				select {
				case e := <-done:
					t.Fatalf("operation completed before lock released: %v", e)
				case <-ctx.Done():
					t.Fatal("operation never waited for row lock")
				case <-time.After(10 * time.Millisecond):
				}
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err = <-done; !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("old password accepted after reset: %v", err)
			}
			if _, _, err = s.Login(ctx, "race_user", "reset password"); err != nil {
				t.Fatalf("reset password lost: %v", err)
			}
		})
	}
}

package auth

import (
	"encoding/base64"
	"fmt"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"
)

func TestUsernameValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"abc", true}, {"A_9", true}, {strings.Repeat("a", 32), true},
		{"", false}, {"ab", false}, {strings.Repeat("a", 33), false}, {"has space", false}, {"a-b", false}, {"中文名", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validName(tc.name); got != tc.valid {
				t.Fatalf("validName(%q)=%v", tc.name, got)
			}
		})
	}
}
func TestDisplayNameValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{{"", true}, {"音乐 🎵", true}, {strings.Repeat("歌", 100), true}, {strings.Repeat("歌", 101), false}, {"a\x00b", false}, {string([]byte{0xff}), false}} {
		if validDisplay(tc.name) != tc.valid {
			t.Fatalf("validDisplay(%q) want %v", tc.name, tc.valid)
		}
	}
}
func TestPasswordBoundariesAndSalts(t *testing.T) {
	for _, tc := range []struct {
		n     int
		valid bool
	}{{7, false}, {8, true}, {1024, true}, {1025, false}} {
		_, err := PasswordHash(strings.Repeat("a", tc.n))
		if (err == nil) != tc.valid {
			t.Fatalf("length %d: %v", tc.n, err)
		}
	}
	a, err := PasswordHash("password one")
	if err != nil {
		t.Fatal(err)
	}
	b, err := PasswordHash("password one")
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !verify("password one", a) || !verify("password one", b) || verify("password two", a) {
		t.Fatal("password hashes must be salted and verified")
	}
	for _, invalid := range []string{"", "bad", "@@$@@", "YQ$Yg"} {
		if verify("password", invalid) {
			t.Fatalf("accepted malformed hash %q", invalid)
		}
	}
}
func TestRateLimiterConcurrentAndExpiry(t *testing.T) {
	limiter := &rateLimiter{entries: make(map[string]rateEntry)}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.allow("client") {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 10 {
		t.Fatalf("accepted %d requests, want 10", accepted.Load())
	}
	if !limiter.allow("other") {
		t.Fatal("different client should have its own limit")
	}
	limiter.mu.Lock()
	limiter.entries["client"] = rateEntry{count: 10, until: time.Now().Add(-time.Second)}
	limiter.mu.Unlock()
	if !limiter.allow("client") {
		t.Fatal("expired limit did not reset")
	}
}

func TestPHCPasswordHash(t *testing.T) {
	hash, err := PasswordHash("correct password")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Fatalf("unexpected hash format %q", hash)
	}
	if valid, upgrade := verifyPassword("correct password", hash); !valid || upgrade {
		t.Fatalf("valid=%v upgrade=%v", valid, upgrade)
	}
	for _, bad := range []string{
		strings.Replace(hash, "argon2id", "argon2i", 1), strings.Replace(hash, "v=19", "v=16", 1),
		strings.Replace(hash, "m=65536", "m=999999", 1), strings.Replace(hash, "m=65536", "m=1", 1),
		strings.Replace(hash, "t=3", "t=0", 1), strings.Replace(hash, "t=3", "t=11", 1),
		strings.Replace(hash, "p=4", "p=0", 1), strings.Replace(hash, "p=4", "p=9", 1),
		hash + "$extra", strings.Repeat("A", 513),
	} {
		if valid, _ := verifyPassword("correct password", bad); valid {
			t.Errorf("accepted invalid hash %q", bad)
		}
	}
}

func TestPHCParameterUpgrade(t *testing.T) {
	salt := []byte("0123456789abcdef")
	for _, tc := range []struct {
		name               string
		memory, iterations uint32
		parallelism        uint8
		upgrade            bool
	}{
		{"older settings", 8192, 1, 1, true},
		{"current settings", passwordMemory, passwordIterations, passwordParallelism, false},
		{"higher memory", 128 * 1024, passwordIterations, passwordParallelism, false},
		{"higher iterations", passwordMemory, 4, passwordParallelism, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sum := argon2.IDKey([]byte("password example"), salt, tc.iterations, tc.memory, tc.parallelism, 32)
			encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", tc.memory, tc.iterations, tc.parallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(sum))
			valid, upgrade := verifyPassword("password example", encoded)
			if !valid || upgrade != tc.upgrade {
				t.Fatalf("valid=%v upgrade=%v", valid, upgrade)
			}
		})
	}
}

func TestLimiterRespectsTrustedProxies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name      string
		trusted   []string
		forwarded bool
	}{
		{"trusted proxy", []string{"192.0.2.1"}, true},
		{"untrusted proxy", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			if err := router.SetTrustedProxies(tc.trusted); err != nil {
				t.Fatal(err)
			}
			h := handler{limiter: &rateLimiter{entries: make(map[string]rateEntry)}}
			router.GET("/", h.limit, func(c *gin.Context) { c.Status(204) })
			request := func(ip string) int {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.0.2.1:1234"
				req.Header.Set("X-Forwarded-For", ip)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				return w.Code
			}
			for i := 0; i < 10; i++ {
				if got := request("198.51.100.1"); got != 204 {
					t.Fatalf("request %d: %d", i, got)
				}
			}
			if got := request("198.51.100.1"); got != 429 {
				t.Fatalf("limit: %d", got)
			}
			got := request("198.51.100.2")
			want := 429
			if tc.forwarded {
				want = 204
			}
			if got != want {
				t.Fatalf("other forwarded client got %d want %d", got, want)
			}
		})
	}
}

package auth

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type handler struct {
	service *Service
	limiter *rateLimiter
}
type rateEntry struct {
	count int
	until time.Time
}
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
}

func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if len(r.entries) >= 10000 {
		for k, v := range r.entries {
			if !v.until.After(now) {
				delete(r.entries, k)
			}
		}
		if len(r.entries) >= 10000 {
			return false
		}
	}
	e := r.entries[key]
	if !e.until.After(now) {
		e = rateEntry{until: now.Add(time.Minute)}
	}
	e.count++
	r.entries[key] = e
	return e.count <= 10
}
func (h handler) limit(c *gin.Context) {
	if !h.limiter.allow(c.ClientIP()) {
		c.Header("Retry-After", "60")
		c.AbortWithStatusJSON(429, gin.H{"error": "rate_limited"})
		return
	}
	c.Next()
}

type input struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func RegisterRoutes(api *gin.RouterGroup, service *Service) {
	h := handler{service: service, limiter: &rateLimiter{entries: make(map[string]rateEntry)}}
	group := api.Group("/auth")
	group.POST("/guest", h.limit, h.guest)
	group.POST("/register", h.limit, h.register)
	group.POST("/login", h.limit, h.login)
	protected := group.Group("")
	protected.Use(h.authenticate)
	protected.GET("/me", h.me)
	protected.PATCH("/me", h.profile)
	protected.POST("/upgrade", h.upgrade)
	protected.POST("/logout", h.logout)
	protected.POST("/logout-all", h.logoutAll)
	protected.POST("/password", h.password)
}
func bind(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	if e := c.ShouldBindJSON(v); e != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return false
	}
	return true
}
func fail(c *gin.Context, e error) {
	switch {
	case errors.Is(e, ErrInvalid):
		c.JSON(400, gin.H{"error": "invalid_request"})
	case errors.Is(e, ErrUnauthorized):
		c.JSON(401, gin.H{"error": "unauthorized"})
	case errors.Is(e, ErrConflict):
		c.JSON(409, gin.H{"error": "username_taken"})
	default:
		c.JSON(500, gin.H{"error": "internal_server_error"})
	}
}
func result(c *gin.Context, u User, t string) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"user": u, "token": t, "token_type": "Bearer", "expires_in": int(lifetime.Seconds())})
}
func (h handler) guest(c *gin.Context) {
	u, t, e := h.service.Create(c.Request.Context(), "", "", "", true)
	if e != nil {
		fail(c, e)
		return
	}
	result(c, u, t)
}
func (h handler) register(c *gin.Context) {
	var v input
	if !bind(c, &v) {
		return
	}
	u, t, e := h.service.Create(c.Request.Context(), v.Username, v.Password, v.DisplayName, false)
	if e != nil {
		fail(c, e)
		return
	}
	result(c, u, t)
}
func (h handler) login(c *gin.Context) {
	var v input
	if !bind(c, &v) {
		return
	}
	u, t, e := h.service.Login(c.Request.Context(), v.Username, v.Password)
	if e != nil {
		fail(c, e)
		return
	}
	result(c, u, t)
}
func (h handler) authenticate(c *gin.Context) {
	a := strings.Split(c.GetHeader("Authorization"), " ")
	if len(a) != 2 || !strings.EqualFold(a[0], "Bearer") {
		fail(c, ErrUnauthorized)
		c.Abort()
		return
	}
	active := c.Request.Method != http.MethodGet || c.FullPath() != "/api/v1/auth/me"
	u, e := h.service.Authenticate(c.Request.Context(), a[1], active)
	if e != nil {
		fail(c, e)
		c.Abort()
		return
	}
	c.Set("user", u)
	c.Set("token", a[1])
	c.Header("Cache-Control", "no-store")
	c.Next()
}
func user(c *gin.Context) User      { return c.MustGet("user").(User) }
func (h handler) me(c *gin.Context) { c.JSON(200, gin.H{"user": user(c)}) }
func (h handler) profile(c *gin.Context) {
	var v struct {
		Username    *string `json:"username"`
		DisplayName *string `json:"display_name"`
	}
	if !bind(c, &v) {
		return
	}
	if e := h.service.Profile(c.Request.Context(), user(c).ID, v.Username, v.DisplayName); e != nil {
		fail(c, e)
		return
	}
	c.Status(204)
}
func (h handler) upgrade(c *gin.Context) {
	var v input
	if !bind(c, &v) {
		return
	}
	if e := h.service.Upgrade(c.Request.Context(), user(c).ID, v.Username, v.Password); e != nil {
		fail(c, e)
		return
	}
	c.Status(204)
}
func (h handler) logout(c *gin.Context) {
	if e := h.service.Logout(c.Request.Context(), user(c).ID, c.GetString("token"), false); e != nil {
		fail(c, e)
		return
	}
	c.Status(204)
}
func (h handler) logoutAll(c *gin.Context) {
	if e := h.service.Logout(c.Request.Context(), user(c).ID, c.GetString("token"), true); e != nil {
		fail(c, e)
		return
	}
	c.Status(204)
}
func (h handler) password(c *gin.Context) {
	var v input
	if !bind(c, &v) {
		return
	}
	if e := h.service.ChangePassword(c.Request.Context(), user(c).ID, v.OldPassword, v.NewPassword); e != nil {
		fail(c, e)
		return
	}
	c.Status(204)
}

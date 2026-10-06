package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/utils"
	"golang.org/x/time/rate"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-XSS-Protection", "0")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'none'")
		c.Next()
	}
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type LoginRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*ipLimiter
	limit    rate.Limit
	burst    int
}

func NewLoginRateLimiter(requests int, window time.Duration) *LoginRateLimiter {
	perSecond := rate.Limit(float64(requests) / window.Seconds())
	if perSecond <= 0 {
		perSecond = 1
	}
	l := &LoginRateLimiter{
		visitors: make(map[string]*ipLimiter),
		limit:    perSecond,
		burst:    requests,
	}
	go l.cleanup()
	return l
}

func (l *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l.mu.Lock()
		v, ok := l.visitors[ip]
		if !ok {
			v = &ipLimiter{limiter: rate.NewLimiter(l.limit, l.burst)}
			l.visitors[ip] = v
		}
		v.lastSeen = time.Now()
		allow := v.limiter.Allow()
		l.mu.Unlock()
		if !allow {
			utils.JSONError(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many login attempts. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}

package demo

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Delay     time.Duration
	ErrorRate float64
}

func FromEnv() (Config, error) {
	var cfg Config
	if value := strings.TrimSpace(os.Getenv("DEMO_DELAY")); value != "" {
		delay, err := time.ParseDuration(value)
		if err != nil || delay < 0 {
			return cfg, errors.New("DEMO_DELAY must be a non-negative duration, e.g. 0s or 1s")
		}
		cfg.Delay = delay
	}
	if value := strings.TrimSpace(os.Getenv("DEMO_ERROR_RATE")); value != "" {
		rate, err := strconv.ParseFloat(value, 64)
		if err != nil || !(rate >= 0 && rate <= 1) {
			return cfg, errors.New("DEMO_ERROR_RATE must be a finite number between 0 and 1")
		}
		cfg.ErrorRate = rate
	}
	return cfg, nil
}

func Middleware(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Delay > 0 {
			timer := time.NewTimer(cfg.Delay)
			defer timer.Stop()
			select {
			case <-c.Request.Context().Done():
				c.AbortWithStatus(http.StatusRequestTimeout)
				return
			case <-timer.C:
			}
		}
		if cfg.ErrorRate > 0 && rand.Float64() < cfg.ErrorRate {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "demo: service unavailable"})
			return
		}
		c.Next()
	}
}

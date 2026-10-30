package prober

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	URL        string
	ExpectedID uuid.UUID
	Interval   time.Duration
	Timeout    time.Duration
	ListenAddr string
}

func NewConfig() (*Config, error) {
	const defaultInterval = 5 * time.Second
	const defaultTimeout = 2 * time.Second
	cfg := &Config{URL: os.Getenv("PROBE_URL"), Interval: defaultInterval, Timeout: defaultTimeout}
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return nil, errors.New("PROBE_URL must be an HTTP(S) URL without credentials or a fragment")
	}
	cfg.ExpectedID, err = uuid.Parse(os.Getenv("PROBE_EXPECTED_WISHLIST_ID"))
	if err != nil || cfg.ExpectedID == uuid.Nil {
		return nil, errors.New("PROBE_EXPECTED_WISHLIST_ID must be a non-zero UUID")
	}
	port := strings.TrimSpace(os.Getenv("PROBE_METRICS_PORT"))
	if port == "" {
		port = "9115"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, errors.New("PROBE_METRICS_PORT must be an integer between 1 and 65535")
	}
	cfg.ListenAddr = ":" + port
	return cfg, nil
}

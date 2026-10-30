package prober

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  map[string]string
		err  string
	}{
		{name: "defaults"},
		{name: "custom metrics port", env: map[string]string{"PROBE_METRICS_PORT": "9116"}},
		{name: "missing URL", env: map[string]string{"PROBE_URL": ""}, err: "PROBE_URL"},
		{name: "invalid scheme", env: map[string]string{"PROBE_URL": "file:///tmp/example"}, err: "PROBE_URL"},
		{name: "credentials", env: map[string]string{"PROBE_URL": "http://user:password@example.test/"}, err: "PROBE_URL"},
		{name: "missing id", env: map[string]string{"PROBE_EXPECTED_WISHLIST_ID": ""}, err: "PROBE_EXPECTED_WISHLIST_ID"},
		{name: "zero id", env: map[string]string{"PROBE_EXPECTED_WISHLIST_ID": "00000000-0000-0000-0000-000000000000"}, err: "PROBE_EXPECTED_WISHLIST_ID"},
		{name: "invalid port", env: map[string]string{"PROBE_METRICS_PORT": "abc"}, err: "PROBE_METRICS_PORT"},
		{name: "zero port", env: map[string]string{"PROBE_METRICS_PORT": "0"}, err: "PROBE_METRICS_PORT"},
		{name: "port out of range", env: map[string]string{"PROBE_METRICS_PORT": "65536"}, err: "PROBE_METRICS_PORT"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROBE_URL", "http://example.test/api/v1/public/token")
			t.Setenv("PROBE_EXPECTED_WISHLIST_ID", "00000000-0000-4000-8000-000000000001")
			t.Setenv("PROBE_METRICS_PORT", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg, err := NewConfig()
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				assert.NotContains(t, err.Error(), "password")
				return
			}
			require.NoError(t, err)
			wantAddr := ":9115"
			if tt.name == "custom metrics port" {
				wantAddr = ":9116"
			}
			assert.Equal(t, wantAddr, cfg.ListenAddr)
		})
	}
}

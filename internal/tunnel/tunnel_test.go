package tunnel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNgrok_SetsFields(t *testing.T) {
	t.Parallel()

	tun := NewNgrok("test-token", "test-domain.ngrok.io")

	assert.NotNil(t, tun)
	assert.Equal(t, "test-token", tun.authToken)
	assert.Equal(t, "test-domain.ngrok.io", tun.domain)
}

func TestNewNgrok_EmptyDomain(t *testing.T) {
	t.Parallel()

	tun := NewNgrok("test-token", "")

	assert.NotNil(t, tun)
	assert.Equal(t, "test-token", tun.authToken)
	assert.Empty(t, tun.domain)
}

func TestNgrokTunnel_PublicURL_BeforeStart(t *testing.T) {
	t.Parallel()

	tun := NewNgrok("test-token", "")

	assert.Empty(t, tun.PublicURL())
}

func TestNgrokTunnel_Close_BeforeStart(t *testing.T) {
	t.Parallel()

	tun := NewNgrok("test-token", "")

	err := tun.Close()
	assert.NoError(t, err, "closing unstarted tunnel should not error")
}

func TestNgrokTunnel_Start_EmptyToken(t *testing.T) {
	t.Parallel()

	tun := NewNgrok("", "")
	_, err := tun.Start(context.Background(), "127.0.0.1:8420")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth token is required")
}

// Note: We do NOT test actual ngrok connection here as that requires a real token
// and would make network calls. Integration tests with real ngrok should be done
// separately if needed, but are not required for basic functionality.

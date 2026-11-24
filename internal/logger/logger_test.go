package logger_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/logger"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		env       string
		wantJSON  bool
		wantDebug bool
	}{
		{env: logger.EnvLocal, wantDebug: true},
		{env: logger.EnvProd, wantJSON: true},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log, err := logger.New(tt.env, &buf)
			require.NoError(t, err)

			log.Debug("dbg")
			log.Info("hello", "k", "v")

			out := buf.String()
			assert.Contains(t, out, "hello")
			assert.Equal(t, tt.wantDebug, bytes.Contains(buf.Bytes(), []byte("dbg")))
			assert.Equal(t, tt.wantJSON, bytes.HasPrefix(buf.Bytes(), []byte("{")))
		})
	}
}

func TestNew_UnknownEnv_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := logger.New("production", &bytes.Buffer{})
	require.ErrorContains(t, err, `unknown environment "production"`)
}

package logging_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/logging"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "returns logger instance"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := logging.New()
			require.NotNil(t, logger)
			assert.NotNil(t, logger.Handler())
		})
	}
}

package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/service"
)

func TestErrorFormattingAndUnwrap(t *testing.T) {
	base := errors.New("boom")

	tests := []struct {
		name       string
		err        *service.Error
		wantText   string
		wantIsBase bool
	}{
		{
			name:       "formats wrapped error",
			err:        &service.Error{Code: 502, Message: "backend failed", Err: base},
			wantText:   "backend failed: boom",
			wantIsBase: true,
		},
		{
			name:     "formats plain message",
			err:      &service.Error{Code: 404, Message: "not found"},
			wantText: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantText, tc.err.Error())
			assert.Equal(t, tc.wantIsBase, errors.Is(tc.err, base))
		})
	}
}

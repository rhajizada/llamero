package xslices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/xslices"
)

func TestUniqueTrimmedStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "trims de-duplicates and drops empty values",
			input: []string{" alpha ", "beta", "alpha", "", "beta", "gamma"},
			want:  []string{"alpha", "beta", "gamma"},
		},
		{
			name:  "returns empty slice for only empty values",
			input: []string{" ", ""},
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, xslices.UniqueTrimmedStrings(tc.input))
		})
	}
}

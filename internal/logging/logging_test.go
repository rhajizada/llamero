package logging_test

import (
	"testing"

	"github.com/rhajizada/llamero/internal/logging"
)

func TestNew(t *testing.T) {
	t.Parallel()

	if logger := logging.New(); logger == nil {
		t.Fatal("expected logger")
	}
}

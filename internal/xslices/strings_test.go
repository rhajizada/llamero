package xslices_test

import (
	"reflect"
	"testing"

	"github.com/rhajizada/llamero/internal/xslices"
)

func TestUniqueTrimmedStrings(t *testing.T) {
	t.Parallel()

	got := xslices.UniqueTrimmedStrings([]string{" alpha ", "beta", "alpha", "", "beta", "gamma"})
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected unique values: got %#v want %#v", got, want)
	}
}

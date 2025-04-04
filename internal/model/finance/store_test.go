package finance

import (
	"github.com/go-bumbu/testdbs"
	"os"
	"testing"
)

// TestMain modifies how test are run,
// it makes sure that the needed DBs are ready and does cleanup in the end.
func TestMain(m *testing.M) {
	testdbs.InitDBS()
	// main block that runs tests
	code := m.Run()
	testdbs.Clean()
	os.Exit(code)
}

const (
	tenant1 = "tenant1"
	tenant2 = "tenant2"
)

func ptr[T any](v T) *T {
	return &v
}

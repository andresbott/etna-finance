package finance

import (
	"fmt"
	"github.com/go-bumbu/testdbs"
	"os"
	"testing"
	"time"
)

// TestMain modifies how test are run,
// it makes sure that the needed DBs are ready and does cleanup in the end.
func TestMain(m *testing.M) {
	testdbs.InitDBS()
	// main block that runs tests
	code := m.Run()
	_ = testdbs.Clean()
	os.Exit(code)
}

const (
	tenant1     = "tenant1"
	tenant2     = "tenant2"
	emptyTenant = "tenantEmpty"
)

// returns a pointer to a specific type
func ptr[T any](v T) *T {
	return &v
}
func getDate(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))
	}
	return parsedTime
}
func getDateTime(timeStr string) time.Time {
	// Parse the string based on the provided layout
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(fmt.Errorf("unable to parse time: %v", err))

	}
	return parsedTime
}

package importer

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IsRateLimit429 reports whether err is an HTTP 429 Too Many Requests (rate limit).
func IsRateLimit429(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "429")
}

// Default429RetryAfter is the default wait time when the API does not return a Retry-After value.
// Matches typical "requests per minute" limits.
const Default429RetryAfter = 60 * time.Second

// RetryAfterFrom429Err returns the suggested wait duration from a 429 error message.
// It looks for a number of seconds in the error text (e.g. "retry after 45", "wait 60 seconds");
// if none is found, returns defaultWait.
func RetryAfterFrom429Err(err error, defaultWait time.Duration) time.Duration {
	if err == nil || defaultWait <= 0 {
		return defaultWait
	}
	s := err.Error()
	// Try "retry after N" or "wait N seconds" or "Retry-After: N"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)retry\s+after\s+(\d+)`),
		regexp.MustCompile(`(?i)wait\s+(\d+)\s*sec`),
		regexp.MustCompile(`(?i)retry-after\s*:\s*(\d+)`),
		regexp.MustCompile(`(\d+)\s*sec`),
	}
	for _, re := range patterns {
		if m := re.FindStringSubmatch(s); len(m) >= 2 {
			if n, e := strconv.Atoi(m[1]); e == nil && n > 0 && n <= 3600 {
				return time.Duration(n) * time.Second
			}
		}
	}
	return defaultWait
}

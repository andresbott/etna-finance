package importer

import (
	"errors"
	"testing"
	"time"
)

func TestIsRateLimit429(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("connection refused"), false},
		{"contains 429", errors.New("HTTP 429 Too Many Requests"), true},
		{"just 429", errors.New("429"), true},
		{"status code 429 in message", errors.New("status: 429, retry later"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimit429(tt.err)
			if got != tt.want {
				t.Errorf("IsRateLimit429(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryAfterFrom429Err(t *testing.T) {
	defaultWait := 60 * time.Second

	tests := []struct {
		name        string
		err         error
		defaultWait time.Duration
		want        time.Duration
	}{
		{"nil error returns default", nil, defaultWait, defaultWait},
		{"negative default returns negative", errors.New("429"), -1 * time.Second, -1 * time.Second},
		{"zero default returns zero", errors.New("429"), 0, 0},
		{"no number returns default", errors.New("rate limited"), defaultWait, defaultWait},
		{"retry after N", errors.New("retry after 45"), defaultWait, 45 * time.Second},
		{"Retry After uppercase", errors.New("Retry After 30"), defaultWait, 30 * time.Second},
		{"wait N seconds", errors.New("please wait 20 seconds"), defaultWait, 20 * time.Second},
		{"Retry-After header style", errors.New("Retry-After: 15"), defaultWait, 15 * time.Second},
		{"N sec pattern", errors.New("try again in 10 sec"), defaultWait, 10 * time.Second},
		{"number over 3600 returns default", errors.New("retry after 7200"), defaultWait, defaultWait},
		{"zero seconds returns default", errors.New("retry after 0"), defaultWait, defaultWait},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RetryAfterFrom429Err(tt.err, tt.defaultWait)
			if got != tt.want {
				t.Errorf("RetryAfterFrom429Err(%v, %v) = %v, want %v", tt.err, tt.defaultWait, got, tt.want)
			}
		})
	}
}

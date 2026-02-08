package main

import (
	"testing"
	"time"
)

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := humanReadableSize(tt.size)
		if result != tt.expected {
			t.Errorf("humanReadableSize(%d) = %q, want %q", tt.size, result, tt.expected)
		}
	}
}

func TestHumanReadableDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{3661 * time.Second, "1h1m1s"},
		{90061 * time.Second, "1d1h1m1s"},
	}

	for _, tt := range tests {
		result := humanReadableDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("humanReadableDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
		}
	}
}

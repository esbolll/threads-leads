package threadsapi

import (
	"testing"
	"time"
)

func TestNormalizeTime(t *testing.T) {
	got := normalizeTime("2026-11-06T07:07:31+0000")
	if got != "2026-11-06T07:07:31Z" {
		t.Fatalf("got %s", got)
	}
	if normalizeTime("garbage") != "garbage" {
		t.Fatal("unknown format must pass through")
	}
}

func TestMinInterval(t *testing.T) {
	// 100 queries per pass must not exceed 2200 per day.
	iv := MinInterval(100)
	if passes := int(24 * time.Hour / iv); passes*100 > QueriesPerDay {
		t.Fatalf("interval %s allows %d queries/day", iv, passes*100)
	}
	if MinInterval(0) != 0 {
		t.Fatal("zero queries need no interval")
	}
}

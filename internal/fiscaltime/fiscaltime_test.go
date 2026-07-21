package fiscaltime_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaltime"
)

func TestNormalizeIssuedAcceptsLuandaPlusOne(t *testing.T) {
	got, err := fiscaltime.NormalizeIssued("2026-07-21T10:00:00+01:00", fiscaltime.AfricaLuanda)
	if err != nil {
		t.Fatal(err)
	}
	wantUTC := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	if !got.InstantUTC.Equal(wantUTC) {
		t.Fatalf("utc=%v want %v", got.InstantUTC, wantUTC)
	}
	if got.UTCString != "2026-07-21T09:00:00.000000Z" {
		t.Fatalf("UTCString=%q", got.UTCString)
	}
	if got.Timezone != fiscaltime.AfricaLuanda || got.OffsetMinutes != 60 {
		t.Fatalf("tz=%q offset=%d", got.Timezone, got.OffsetMinutes)
	}
}

func TestNormalizeIssuedRejectsZForLuanda(t *testing.T) {
	_, err := fiscaltime.NormalizeIssued("2026-07-21T09:00:00Z", fiscaltime.AfricaLuanda)
	if !errors.Is(err, fiscaltime.ErrInvalidIssuedAt) {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeIssuedRejectsIncompatibleOffset(t *testing.T) {
	_, err := fiscaltime.NormalizeIssued("2026-07-21T10:00:00+02:00", fiscaltime.AfricaLuanda)
	if !errors.Is(err, fiscaltime.ErrInvalidIssuedAt) {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeIssuedRejectsMissingOffset(t *testing.T) {
	_, err := fiscaltime.NormalizeIssued("2026-07-21T10:00:00", fiscaltime.AfricaLuanda)
	if !errors.Is(err, fiscaltime.ErrInvalidIssuedAt) {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeIssuedIndependentOfProcessTZ(t *testing.T) {
	prev := os.Getenv("TZ")
	t.Cleanup(func() { _ = os.Setenv("TZ", prev) })
	if err := os.Setenv("TZ", "America/New_York"); err != nil {
		t.Fatal(err)
	}
	time.Local = time.FixedZone("test-local", -5*3600)

	got, err := fiscaltime.NormalizeIssued("2026-07-21T10:00:00.123456+01:00", fiscaltime.AfricaLuanda)
	if err != nil {
		t.Fatal(err)
	}
	if got.UTCString != "2026-07-21T09:00:00.123456Z" {
		t.Fatalf("UTCString=%q under TZ=America/New_York", got.UTCString)
	}
	if got.OffsetMinutes != 60 {
		t.Fatalf("offset=%d", got.OffsetMinutes)
	}
}

func TestRebuildLocalLuanda(t *testing.T) {
	utc := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	local, err := fiscaltime.RebuildLocal(utc, fiscaltime.AfricaLuanda)
	if err != nil {
		t.Fatal(err)
	}
	_, off := local.Zone()
	if off != 3600 {
		t.Fatalf("offset sec=%d", off)
	}
	if local.Hour() != 10 || local.Day() != 21 {
		t.Fatalf("local=%v", local)
	}
}

func TestValidateNormalizedContextOK(t *testing.T) {
	err := fiscaltime.ValidateNormalizedContext("2026-07-21T09:00:00.000000Z", fiscaltime.AfricaLuanda, 60)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateNormalizedContextRejects(t *testing.T) {
	cases := []struct {
		name   string
		utc    string
		tz     string
		offset int
	}{
		{"unknown_tz", "2026-07-21T09:00:00.000000Z", "Not/ARealZone", 60},
		{"incompatible_offset", "2026-07-21T09:00:00.000000Z", fiscaltime.AfricaLuanda, 0},
		{"non_canonical_z", "2026-07-21T09:00:00Z", fiscaltime.AfricaLuanda, 60},
		{"offset_string", "2026-07-21T10:00:00+01:00", fiscaltime.AfricaLuanda, 60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := fiscaltime.ValidateNormalizedContext(tc.utc, tc.tz, tc.offset)
			if !errors.Is(err, fiscaltime.ErrInvalidIssuedAt) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

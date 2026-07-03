package convert_test

import (
	"testing"
	"time"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

const (
	testLocal                 = "local"
	testUploaded              = "uploaded"
	testFileSourceUnspecified = "FILE_SOURCE_UNSPECIFIED"
	testLocalOnly             = "local_only"
	testFileStatusUnspecified = "FILE_STATUS_UNSPECIFIED"
)

func TestPTime_FromZero(t *testing.T) {
	t.Parallel()
	result := convert.PTime(time.Time{})
	if result != nil {
		t.Errorf("PTime(zero) = %v, want nil", result)
	}
}

func TestPTime_FromNonZero(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	result := convert.PTime(now)
	if result == nil {
		t.Fatal("PTime(nonzero) = nil, want timestamp")
	}
	if got := result.AsTime(); !got.Equal(now) {
		t.Errorf("PTime(nonzero) = %v, want %v", got, now)
	}
}

func TestPTimeToTime_Nil(t *testing.T) {
	t.Parallel()
	result := convert.PTimeToTime(nil)
	if !result.IsZero() {
		t.Errorf("PTimeToTime(nil) = %v, want zero time", result)
	}
}

func TestFileSourceToProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{testLocal, "FILE_SOURCE_LOCAL"},
		{testUploaded, "FILE_SOURCE_UPLOADED"},
		{"synced", "FILE_SOURCE_SYNCED"},
		{"unknown", testFileSourceUnspecified},
		{"", testFileSourceUnspecified},
	}
	for _, tt := range tests {
		got := convert.FileSourceToProto(tt.input)
		if got != tt.want {
			t.Errorf("FileSourceToProto(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileSourceToEnt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"FILE_SOURCE_LOCAL", testLocal},
		{"FILE_SOURCE_UPLOADED", testUploaded},
		{"FILE_SOURCE_SYNCED", "synced"},
		{"FILE_SOURCE_UNSPECIFIED", testLocal},
		{"SOMETHING_ELSE", testLocal},
		{"", testLocal},
	}
	for _, tt := range tests {
		got := convert.FileSourceToEnt(tt.input)
		if got != tt.want {
			t.Errorf("FileSourceToEnt(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileStatusToProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{testLocalOnly, "FILE_STATUS_LOCAL_ONLY"},
		{testUploaded, "FILE_STATUS_UPLOADED"},
		{"downloaded", "FILE_STATUS_DOWNLOADED"},
		{"cloud_only", "FILE_STATUS_CLOUD_ONLY"},
		{"unknown", testFileStatusUnspecified},
		{"", testFileStatusUnspecified},
	}
	for _, tt := range tests {
		got := convert.FileStatusToProto(tt.input)
		if got != tt.want {
			t.Errorf("FileStatusToProto(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileStatusToEnt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"FILE_STATUS_LOCAL_ONLY", testLocalOnly},
		{"FILE_STATUS_UPLOADED", testUploaded},
		{"FILE_STATUS_DOWNLOADED", "downloaded"},
		{"FILE_STATUS_CLOUD_ONLY", "cloud_only"},
		{"FILE_STATUS_UNSPECIFIED", testLocalOnly},
		{"", testLocalOnly},
	}
	for _, tt := range tests {
		got := convert.FileStatusToEnt(tt.input)
		if got != tt.want {
			t.Errorf("FileStatusToEnt(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPTimeSafe_Nil(t *testing.T) {
	t.Parallel()
	result := convert.PTimeSafe(nil)
	if result != nil {
		t.Errorf("PTimeSafe(nil) = %v, want nil", result)
	}
}

func TestPTimeSafe_ZeroTime(t *testing.T) {
	t.Parallel()
	zero := time.Time{}
	result := convert.PTimeSafe(&zero)
	if result != nil {
		t.Errorf("PTimeSafe(zero) = %v, want nil", result)
	}
}

func TestPTimeSafe_NonZero(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	result := convert.PTimeSafe(&now)
	if result == nil {
		t.Fatal("PTimeSafe(nonzero) = nil, want timestamp")
	}
	if got := result.AsTime(); !got.Equal(now) {
		t.Errorf("PTimeSafe() = %v, want %v", got, now)
	}
}

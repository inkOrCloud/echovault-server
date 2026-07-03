// Package convert provides helper functions for converting between proto and ent types.
package convert

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	sourceLocal     = "local"
	sourceUploaded  = "uploaded"
	sourceSynced    = "synced"
	statusLocalOnly = "local_only"
)

// PTime converts a time.Time to a protobuf Timestamp pointer.
func PTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// PTimeToTime converts a protobuf Timestamp pointer to time.Time.
func PTimeToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// FileSourceToProto converts an ent file source string to proto enum name.
func FileSourceToProto(s string) string {
	switch s {
	case sourceLocal:
		return "FILE_SOURCE_LOCAL"
	case sourceUploaded:
		return "FILE_SOURCE_UPLOADED"
	case sourceSynced:
		return "FILE_SOURCE_SYNCED"
	default:
		return "FILE_SOURCE_UNSPECIFIED"
	}
}

// FileSourceToEnt converts a proto enum name to ent file source string.
func FileSourceToEnt(s string) string {
	switch s {
	case "FILE_SOURCE_LOCAL":
		return sourceLocal
	case "FILE_SOURCE_UPLOADED":
		return sourceUploaded
	case "FILE_SOURCE_SYNCED":
		return sourceSynced
	default:
		return sourceLocal
	}
}

// FileStatusToProto converts an ent file status string to proto enum name.
func FileStatusToProto(s string) string {
	switch s {
	case statusLocalOnly:
		return "FILE_STATUS_LOCAL_ONLY"
	case sourceUploaded:
		return "FILE_STATUS_UPLOADED"
	case "downloaded":
		return "FILE_STATUS_DOWNLOADED"
	case "cloud_only":
		return "FILE_STATUS_CLOUD_ONLY"
	default:
		return "FILE_STATUS_UNSPECIFIED"
	}
}

// FileStatusToEnt converts a proto enum name to ent file status string.
func FileStatusToEnt(s string) string {
	switch s {
	case "FILE_STATUS_LOCAL_ONLY":
		return statusLocalOnly
	case "FILE_STATUS_UPLOADED":
		return sourceUploaded
	case "FILE_STATUS_DOWNLOADED":
		return "downloaded"
	case "FILE_STATUS_CLOUD_ONLY":
		return "cloud_only"
	default:
		return statusLocalOnly
	}
}

// PTimeSafe safely converts a *time.Time to a protobuf Timestamp pointer.
func PTimeSafe(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

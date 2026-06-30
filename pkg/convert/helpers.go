package convert

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func PTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func PTimeToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func FileSourceToProto(s string) string {
	switch s {
	case "local":
		return "FILE_SOURCE_LOCAL"
	case "uploaded":
		return "FILE_SOURCE_UPLOADED"
	case "synced":
		return "FILE_SOURCE_SYNCED"
	default:
		return "FILE_SOURCE_UNSPECIFIED"
	}
}

func FileSourceToEnt(s string) string {
	switch s {
	case "FILE_SOURCE_LOCAL":
		return "local"
	case "FILE_SOURCE_UPLOADED":
		return "uploaded"
	case "FILE_SOURCE_SYNCED":
		return "synced"
	default:
		return "local"
	}
}

func FileStatusToProto(s string) string {
	switch s {
	case "local_only":
		return "FILE_STATUS_LOCAL_ONLY"
	case "uploaded":
		return "FILE_STATUS_UPLOADED"
	case "downloaded":
		return "FILE_STATUS_DOWNLOADED"
	case "cloud_only":
		return "FILE_STATUS_CLOUD_ONLY"
	default:
		return "FILE_STATUS_UNSPECIFIED"
	}
}

func FileStatusToEnt(s string) string {
	switch s {
	case "FILE_STATUS_LOCAL_ONLY":
		return "local_only"
	case "FILE_STATUS_UPLOADED":
		return "uploaded"
	case "FILE_STATUS_DOWNLOADED":
		return "downloaded"
	case "FILE_STATUS_CLOUD_ONLY":
		return "cloud_only"
	default:
		return "local_only"
	}
}

func PTimeSafe(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

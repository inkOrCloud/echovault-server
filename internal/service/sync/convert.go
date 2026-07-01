package sync

import (
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

func EntToProto(s *ent.SyncLog) *syncpb.SyncChange {
	if s == nil {
		return nil
	}
	return &syncpb.SyncChange{
		EntityType: s.EntityType,
		EntityId:   s.EntityID,
		Action:     actionToProto(s.Action),
		Version:    s.Version,
		Data:       s.Data,
		Timestamp:  convert.PTime(s.Timestamp),
		DeviceId:   s.DeviceID,
	}
}

func actionToProto(s string) syncpb.SyncChange_Action {
	switch s {
	case "create":
		return syncpb.SyncChange_ACTION_CREATE
	case "update":
		return syncpb.SyncChange_ACTION_UPDATE
	case "delete":
		return syncpb.SyncChange_ACTION_DELETE
	default:
		return syncpb.SyncChange_ACTION_UNSPECIFIED
	}
}

func ProtoActionToEnt(pb syncpb.SyncChange_Action) string {
	switch pb {
	case syncpb.SyncChange_ACTION_CREATE:
		return "create"
	case syncpb.SyncChange_ACTION_UPDATE:
		return "update"
	case syncpb.SyncChange_ACTION_DELETE:
		return "delete"
	default:
		return "unknown"
	}
}

package sync

import (
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

type Resolution int

const (
	ResolutionServerWins Resolution = iota + 1
	ResolutionDuplicateKept
)

type ConflictInfo struct {
	EntityType    string
	EntityID      string
	LocalVersion  int64
	ServerVersion int64
	Resolution    Resolution
}

type ConflictResolver struct{}

func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

func (r *ConflictResolver) Resolve(local []*syncpb.SyncChange, getCurrentVersion func(entityType, entityID string) int64) []ConflictInfo {
	var conflicts []ConflictInfo
	for _, lc := range local {
		sv := getCurrentVersion(lc.EntityType, lc.EntityId)
		if sv > 0 && lc.Version < sv {
			conflicts = append(conflicts, ConflictInfo{
				EntityType:    lc.EntityType,
				EntityID:      lc.EntityId,
				LocalVersion:  lc.Version,
				ServerVersion: sv,
				Resolution:    ResolutionServerWins,
			})
		}
	}
	return conflicts
}

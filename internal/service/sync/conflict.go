// Package sync provides sync and conflict resolution.
package sync

import (
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

// Resolution indicates how a sync conflict was resolved.
type Resolution int

// Resolution indicates how a sync conflict was resolved.
const (
	ResolutionServerWins Resolution = iota + 1
	ResolutionDuplicateKept
)

// ConflictInfo describes a sync conflict.
type ConflictInfo struct {
	EntityType    string
	EntityID      string
	LocalVersion  int64
	ServerVersion int64
	Resolution    Resolution
}

// ConflictResolver detects conflicts between local and server changes.
type ConflictResolver struct{}

// NewConflictResolver creates a new ConflictResolver.
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// Resolve returns conflicts where local version is behind server version.
func (r *ConflictResolver) Resolve(local []*syncpb.SyncChange, getCurrentVersion func(entityType, entityID string) int64) []ConflictInfo {
	var conflicts []ConflictInfo
	for _, lc := range local {
		sv := getCurrentVersion(lc.GetEntityType(), lc.GetEntityId())
		if sv > 0 && lc.GetVersion() < sv {
			conflicts = append(conflicts, ConflictInfo{
				EntityType:    lc.GetEntityType(),
				EntityID:      lc.GetEntityId(),
				LocalVersion:  lc.GetVersion(),
				ServerVersion: sv,
				Resolution:    ResolutionServerWins,
			})
		}
	}
	return conflicts
}

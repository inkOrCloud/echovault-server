package sync_test

import (
	"testing"

	syncsvc "github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

func TestResolve_NoConflict(t *testing.T) {
	r := syncsvc.NewConflictResolver()
	conflicts := r.Resolve([]*syncpb.SyncChange{
		{EntityType: "song", EntityId: "s1", Version: 1},
	}, func(_, _ string) int64 { return 0 })
	if len(conflicts) != 0 {
		t.Errorf("Resolve() conflicts = %d, want 0", len(conflicts))
	}
}

func TestResolve_ServerWins(t *testing.T) {
	r := syncsvc.NewConflictResolver()
	conflicts := r.Resolve([]*syncpb.SyncChange{
		{EntityType: "song", EntityId: "s1", Version: 1},
	}, func(_, _ string) int64 { return 3 })
	if len(conflicts) != 1 {
		t.Fatalf("Resolve() conflicts = %d, want 1", len(conflicts))
	}
	if conflicts[0].Resolution != syncsvc.ResolutionServerWins {
		t.Errorf("Resolution = %v, want ServerWins", conflicts[0].Resolution)
	}
}

func TestResolve_NoConflictWhenSameVersion(t *testing.T) {
	r := syncsvc.NewConflictResolver()
	conflicts := r.Resolve([]*syncpb.SyncChange{
		{EntityType: "song", EntityId: "s1", Version: 5},
	}, func(_, _ string) int64 { return 5 })
	if len(conflicts) != 0 {
		t.Errorf("Resolve() conflicts = %d, want 0 (equal version)", len(conflicts))
	}
}

func TestResolve_MultipleChanges(t *testing.T) {
	r := syncsvc.NewConflictResolver()
	conflicts := r.Resolve([]*syncpb.SyncChange{
		{EntityType: "song", EntityId: "s1", Version: 1},  // conflict
		{EntityType: "song", EntityId: "s2", Version: 5},  // ok
		{EntityType: "playlist", EntityId: "p1", Version: 2}, // conflict
	}, func(_, id string) int64 {
		if id == "s1" { return 3 }
		if id == "p1" { return 5 }
		return 0
	})
	if len(conflicts) != 2 {
		t.Errorf("Resolve() conflicts = %d, want 2", len(conflicts))
	}
}

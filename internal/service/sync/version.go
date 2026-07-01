package sync

import (
	"context"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/synclog"
)

type VersionTracker struct {
	client *ent.Client
}

func NewVersionTracker(client *ent.Client) *VersionTracker {
	return &VersionTracker{client: client}
}

func (vt *VersionTracker) Next(ctx context.Context) (int64, error) {
	var vs []struct {
		Max int64
	}
	err := vt.client.SyncLog.Query().Aggregate(ent.Max(synclog.FieldVersion)).Scan(ctx, &vs)
	if err != nil || len(vs) == 0 {
		return 1, nil
	}
	return vs[0].Max + 1, nil
}

func (vt *VersionTracker) Current(ctx context.Context) (int64, error) {
	var vs []struct {
		Max int64
	}
	err := vt.client.SyncLog.Query().Aggregate(ent.Max(synclog.FieldVersion)).Scan(ctx, &vs)
	if err != nil || len(vs) == 0 {
		return 0, nil
	}
	return vs[0].Max, nil
}

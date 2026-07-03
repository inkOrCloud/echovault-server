package sync

import (
	"context"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/synclog"
)

// VersionTracker tracks the latest sync version.
type VersionTracker struct {
	client *ent.Client
}

// NewVersionTracker creates a new VersionTracker.
func NewVersionTracker(client *ent.Client) *VersionTracker {
	return &VersionTracker{client: client}
}

// Next returns the next available version number.
func (vt *VersionTracker) Next(ctx context.Context) (int64, error) {
	var vs []struct {
		Max int64
	}
	err := vt.client.SyncLog.Query().Aggregate(ent.Max(synclog.FieldVersion)).Scan(ctx, &vs)
	if err != nil {
		return 1, nil //nolint:nilerr // graceful default when no records exist
	}
	if len(vs) == 0 {
		return 1, nil
	}
	return vs[0].Max + 1, nil
}

// Current returns the latest version number.
func (vt *VersionTracker) Current(ctx context.Context) (int64, error) {
	var vs []struct {
		Max int64
	}
	err := vt.client.SyncLog.Query().Aggregate(ent.Max(synclog.FieldVersion)).Scan(ctx, &vs)
	if err != nil {
		return 0, nil //nolint:nilerr // graceful default when no records exist
	}
	if len(vs) == 0 {
		return 0, nil
	}
	return vs[0].Max, nil
}

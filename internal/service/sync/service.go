package sync

import (
	"context"
	"fmt"

	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
)

// PushResponse is the result of pushing local changes.
type PushResponse struct {
	ServerVersion int64
	AcceptedCount int32
	Conflicts     []ConflictInfo
}

// Service manages sync operations.
type Service struct {
	client   *ent.Client
	version  *VersionTracker
	logger   *ChangeLogger
	resolver *ConflictResolver
	notifier *Notifier
}

// NewService creates a new sync Service.
func NewService(client *ent.Client) *Service {
	return &Service{
		client:   client,
		version:  NewVersionTracker(client),
		logger:   NewChangeLogger(client),
		resolver: NewConflictResolver(),
		notifier: NewNotifier(),
	}
}

// PushChanges accepts a batch of changes from a device.
func (s *Service) PushChanges(ctx context.Context, _ string, _ int64, changes []*syncpb.SyncChange) (*PushResponse, error) {
	if len(changes) == 0 {
		cv, _ := s.version.Current(ctx)
		return &PushResponse{ServerVersion: cv}, nil
	}

	getVersion := func(entityType, entityID string) int64 {
		record, err := s.logger.GetByEntity(ctx, entityType, entityID)
		if err != nil || record == nil {
			return 0
		}
		return record.Version
	}

	conflicts := s.resolver.Resolve(changes, getVersion)

	var accepted int32
	for _, change := range changes {
		shouldSkip := false
		for _, c := range conflicts {
			if c.EntityID == change.GetEntityId() && c.Resolution == ResolutionServerWins {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		v, err := s.version.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("next version: %w", err)
		}
		change.Version = v

		err = s.logger.Append(ctx, change)
		if err != nil {
			return nil, fmt.Errorf("log change: %w", err)
		}
		accepted++

		s.notifier.Notify(&syncpb.ChangeNotification{
			EntityType: change.GetEntityType(),
			Action:     change.GetAction().String(),
			NewVersion: v,
		})
	}

	cv, _ := s.version.Current(ctx)
	return &PushResponse{
		ServerVersion: cv,
		AcceptedCount: accepted,
		Conflicts:     conflicts,
	}, nil
}

// PullChanges returns changes since a given version.
func (s *Service) PullChanges(ctx context.Context, _ string, sinceVersion int64, limit int) ([]*ent.SyncLog, error) {
	logs, err := s.logger.QuerySince(ctx, sinceVersion, limit)
	if err != nil {
		return nil, fmt.Errorf("query changes: %w", err)
	}
	return logs, nil
}

// Subscribe returns a channel for real-time change notifications.
func (s *Service) Subscribe(deviceID string) <-chan *syncpb.ChangeNotification {
	return s.notifier.Subscribe(deviceID)
}

// Unsubscribe removes a device from change notifications.
func (s *Service) Unsubscribe(deviceID string) {
	s.notifier.Unsubscribe(deviceID)
}

// AckChanges acknowledges receipt of changes up to a version.
func (s *Service) AckChanges(ctx context.Context, _ string, version int64) error {
	err := s.logger.Ack(ctx, version)
	if err != nil {
		return fmt.Errorf("ack changes: %w", err)
	}
	return nil
}

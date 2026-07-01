package sync

import (
	"context"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

type PushResponse struct {
	ServerVersion int64
	AcceptedCount int32
	Conflicts     []ConflictInfo
}

type Service struct {
	client   *ent.Client
	version  *VersionTracker
	logger   *ChangeLogger
	resolver *ConflictResolver
	notifier *Notifier
}

func NewService(client *ent.Client) *Service {
	return &Service{
		client:   client,
		version:  NewVersionTracker(client),
		logger:   NewChangeLogger(client),
		resolver: NewConflictResolver(),
		notifier: NewNotifier(),
	}
}

func (s *Service) PushChanges(ctx context.Context, deviceID string, lastPullVersion int64, changes []*syncpb.SyncChange) (*PushResponse, error) {
	if len(changes) == 0 {
		cv, _ := s.version.Current(ctx)
		return &PushResponse{ServerVersion: cv}, nil
	}

	// 构建冲突检测函数：查询每个实体的最新版本
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
			if c.EntityID == change.EntityId && c.Resolution == ResolutionServerWins {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		v, err := s.version.Next(ctx)
		if err != nil {
			return nil, err
		}
		change.Version = v

		if err := s.logger.Append(ctx, change); err != nil {
			return nil, err
		}
		accepted++

		s.notifier.Notify(&syncpb.ChangeNotification{
			EntityType: change.EntityType,
			Action:     change.Action.String(),
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

func (s *Service) PullChanges(ctx context.Context, deviceID string, sinceVersion int64, limit int) ([]*ent.SyncLog, error) {
	return s.logger.QuerySince(ctx, sinceVersion, limit)
}

func (s *Service) Subscribe(deviceID string) <-chan *syncpb.ChangeNotification {
	return s.notifier.Subscribe(deviceID)
}

func (s *Service) Unsubscribe(deviceID string) {
	s.notifier.Unsubscribe(deviceID)
}

func (s *Service) AckChanges(ctx context.Context, deviceID string, version int64) error {
	return s.logger.Ack(ctx, version)
}

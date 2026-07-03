package grpc

import (
	"context"

	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultPullBatchSize = 100

// SyncHandler implements the SyncService gRPC server.
type SyncHandler struct {
	syncpb.UnimplementedSyncServiceServer

	svc *sync.Service
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(svc *sync.Service) *SyncHandler {
	return &SyncHandler{svc: svc}
}

// PushChanges pushes device changes to the server.
func (h *SyncHandler) PushChanges(ctx context.Context, req *syncpb.PushChangesRequest) (*syncpb.PushChangesResponse, error) {
	resp, err := h.svc.PushChanges(ctx, req.GetDeviceId(), req.GetLastPullVersion(), req.GetChanges())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}

	pbConflicts := make([]*syncpb.ConflictInfo, len(resp.Conflicts))
	for i, c := range resp.Conflicts {
		resolution := syncpb.ConflictInfo_RESOLUTION_SERVER_WINS
		if c.Resolution == sync.ResolutionDuplicateKept {
			resolution = syncpb.ConflictInfo_RESOLUTION_DUPLICATE_KEPT
		}
		pbConflicts[i] = &syncpb.ConflictInfo{
			EntityType:    c.EntityType,
			EntityId:      c.EntityID,
			LocalVersion:  c.LocalVersion,
			ServerVersion: c.ServerVersion,
			Resolution:    resolution,
		}
	}

	return &syncpb.PushChangesResponse{
		ServerVersion: resp.ServerVersion,
		AcceptedCount: resp.AcceptedCount,
		Conflicts:     pbConflicts,
	}, nil
}

// PullChanges streams changes from the server to the client.
func (h *SyncHandler) PullChanges(req *syncpb.PullChangesRequest, stream syncpb.SyncService_PullChangesServer) error {
	changes, err := h.svc.PullChanges(stream.Context(), req.GetDeviceId(), req.GetSinceVersion(), defaultPullBatchSize)
	if err != nil {
		return status.Error(codes.Internal, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}

	for _, change := range changes {
		pbChange := syncEntToProto(change)
		err := stream.Send(&syncpb.PullChangesResponse{Change: pbChange})
		if err != nil {
			return err //nolint:wrapcheck // gRPC stream errors are propagated directly
		}
	}
	return nil
}

// SubscribeChanges streams real-time change notifications to the client.
func (h *SyncHandler) SubscribeChanges(req *syncpb.SubscribeChangesRequest, stream syncpb.SyncService_SubscribeChangesServer) error {
	ch := h.svc.Subscribe(req.GetDeviceId())
	defer h.svc.Unsubscribe(req.GetDeviceId())

	for {
		select {
		case notif, ok := <-ch:
			if !ok {
				return nil
			}
			err := stream.Send(&syncpb.SubscribeChangesResponse{
				Notification: notif,
			})
			if err != nil {
				return err //nolint:wrapcheck // gRPC stream errors are propagated directly
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// AckChanges acknowledges receipt of changes up to a given version.
func (h *SyncHandler) AckChanges(ctx context.Context, req *syncpb.AckChangesRequest) (*syncpb.AckChangesResponse, error) {
	err := h.svc.AckChanges(ctx, req.GetDeviceId(), req.GetAckedVersion())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &syncpb.AckChangesResponse{}, nil
}

func syncEntToProto(s *ent.SyncLog) *syncpb.SyncChange {
	if s == nil {
		return nil
	}
	action := syncpb.SyncChange_ACTION_UNSPECIFIED
	switch s.Action {
	case "create":
		action = syncpb.SyncChange_ACTION_CREATE
	case "update":
		action = syncpb.SyncChange_ACTION_UPDATE
	case "delete":
		action = syncpb.SyncChange_ACTION_DELETE
	}
	return &syncpb.SyncChange{
		EntityType: s.EntityType,
		EntityId:   s.EntityID,
		Action:     action,
		Version:    s.Version,
		Data:       s.Data,
		Timestamp:  convert.PTime(s.Timestamp),
		DeviceId:   s.DeviceID,
	}
}

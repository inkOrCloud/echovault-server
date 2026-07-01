package grpc

import (
	"context"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SyncHandler struct {
	syncpb.UnimplementedSyncServiceServer
	svc *sync.Service
}

func NewSyncHandler(svc *sync.Service) *SyncHandler {
	return &SyncHandler{svc: svc}
}

func (h *SyncHandler) PushChanges(ctx context.Context, req *syncpb.PushChangesRequest) (*syncpb.PushChangesResponse, error) {
	resp, err := h.svc.PushChanges(ctx, req.DeviceId, req.LastPullVersion, req.Changes)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
		ServerVersion:  resp.ServerVersion,
		AcceptedCount:  resp.AcceptedCount,
		Conflicts:      pbConflicts,
	}, nil
}

func (h *SyncHandler) PullChanges(req *syncpb.PullChangesRequest, stream syncpb.SyncService_PullChangesServer) error {
	changes, err := h.svc.PullChanges(stream.Context(), req.DeviceId, req.SinceVersion, 100)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	for _, change := range changes {
		pbChange := syncEntToProto(change)
		if err := stream.Send(&syncpb.PullChangesResponse{Change: pbChange}); err != nil {
			return err
		}
	}
	return nil
}

func (h *SyncHandler) SubscribeChanges(req *syncpb.SubscribeChangesRequest, stream syncpb.SyncService_SubscribeChangesServer) error {
	ch := h.svc.Subscribe(req.DeviceId)
	defer h.svc.Unsubscribe(req.DeviceId)

	for {
		select {
		case notif, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&syncpb.SubscribeChangesResponse{
				Notification: notif,
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func (h *SyncHandler) AckChanges(ctx context.Context, req *syncpb.AckChangesRequest) (*syncpb.AckChangesResponse, error) {
	if err := h.svc.AckChanges(ctx, req.DeviceId, req.AckedVersion); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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

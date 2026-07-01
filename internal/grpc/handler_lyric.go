package grpc

import (
	"context"

	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LyricHandler struct {
	lyricpb.UnimplementedLyricServiceServer
	svc *lyric.Service
}

func NewLyricHandler(svc *lyric.Service) *LyricHandler {
	return &LyricHandler{svc: svc}
}

func (h *LyricHandler) GetLyric(ctx context.Context, req *lyricpb.GetLyricRequest) (*lyricpb.GetLyricResponse, error) {
	l, err := h.svc.GetLyric(ctx, req.SongId, req.Language, req.Type)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &lyricpb.GetLyricResponse{Lyrics: []*lyricpb.Lyric{l}}, nil
}

func (h *LyricHandler) SaveLyric(ctx context.Context, req *lyricpb.SaveLyricRequest) (*lyricpb.SaveLyricResponse, error) {
	l, err := h.svc.SaveLyric(ctx, req.SongId, req.Content, req.Type, req.Language)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &lyricpb.SaveLyricResponse{Lyric: l}, nil
}

func (h *LyricHandler) DeleteLyric(ctx context.Context, req *lyricpb.DeleteLyricRequest) (*lyricpb.DeleteLyricResponse, error) {
	if err := h.svc.DeleteLyric(ctx, req.SongId, req.Type, req.Language); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &lyricpb.DeleteLyricResponse{}, nil
}

package grpc

import (
	"context"

	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LyricHandler implements the LyricService gRPC server.
type LyricHandler struct {
	lyricpb.UnimplementedLyricServiceServer

	svc *lyric.Service
}

// NewLyricHandler creates a new LyricHandler.
func NewLyricHandler(svc *lyric.Service) *LyricHandler {
	return &LyricHandler{svc: svc}
}

// GetLyric returns lyrics for a song.
func (h *LyricHandler) GetLyric(ctx context.Context, req *lyricpb.GetLyricRequest) (*lyricpb.GetLyricResponse, error) {
	l, err := h.svc.GetLyric(ctx, req.GetSongId(), req.GetLanguage(), req.GetType())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &lyricpb.GetLyricResponse{Lyrics: []*lyricpb.Lyric{l}}, nil
}

// SaveLyric saves lyrics for a song.
func (h *LyricHandler) SaveLyric(ctx context.Context, req *lyricpb.SaveLyricRequest) (*lyricpb.SaveLyricResponse, error) {
	l, err := h.svc.SaveLyric(ctx, req.GetSongId(), req.GetContent(), req.GetType(), req.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &lyricpb.SaveLyricResponse{Lyric: l}, nil
}

// DeleteLyric removes lyrics for a song.
func (h *LyricHandler) DeleteLyric(ctx context.Context, req *lyricpb.DeleteLyricRequest) (*lyricpb.DeleteLyricResponse, error) {
	err := h.svc.DeleteLyric(ctx, req.GetSongId(), req.GetType(), req.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &lyricpb.DeleteLyricResponse{}, nil
}

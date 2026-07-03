package grpc

import (
	"context"

	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errSongNotFound is a sentinel error for missing songs.

// SongHandler implements the SongService gRPC server.
type SongHandler struct {
	songpb.UnimplementedSongServiceServer

	svc *song.Service
}

// NewSongHandler creates a new SongHandler.
func NewSongHandler(svc *song.Service) *SongHandler {
	return &SongHandler{svc: svc}
}

// CheckSongsByHash checks which songs exist by their file hashes.
func (h *SongHandler) CheckSongsByHash(ctx context.Context, req *songpb.CheckSongsByHashRequest) (*songpb.CheckSongsByHashResponse, error) {
	results, err := h.svc.CheckSongsByHash(ctx, req.GetFileHashes())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbResults := make([]*songpb.CheckSongsByHashResponse_Result, len(results))
	for i, r := range results {
		pbResults[i] = &songpb.CheckSongsByHashResponse_Result{
			FileHash: r.FileHash,
			Exists:   r.Exists,
			Song:     r.Song,
		}
	}
	return &songpb.CheckSongsByHashResponse{Results: pbResults}, nil
}

// PublishSong publishes a new song to the server.
func (h *SongHandler) PublishSong(ctx context.Context, req *songpb.PublishSongRequest) (*songpb.PublishSongResponse, error) {
	s, err := h.svc.PublishSong(ctx, req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &songpb.PublishSongResponse{Song: s}, nil
}

// GetSong returns a song by ID.
func (h *SongHandler) GetSong(ctx context.Context, req *songpb.GetSongRequest) (*songpb.GetSongResponse, error) {
	s, err := h.svc.GetSong(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &songpb.GetSongResponse{Song: s}, nil
}

// SearchSongs searches for songs matching the query.
func (h *SongHandler) SearchSongs(ctx context.Context, req *songpb.SearchSongsRequest) (*songpb.SearchSongsResponse, error) {
	songs, err := h.svc.SearchSongs(ctx, req.GetQuery(), int(req.GetPagination().GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &songpb.SearchSongsResponse{Songs: songs}, nil
}

// ListSongs returns a paginated list of songs.
func (h *SongHandler) ListSongs(ctx context.Context, req *songpb.ListSongsRequest) (*songpb.ListSongsResponse, error) {
	limit := int(req.GetPagination().GetPageSize())
	if limit <= 0 {
		limit = 20
	}
	songs, err := h.svc.ListSongs(ctx, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &songpb.ListSongsResponse{Songs: songs}, nil
}

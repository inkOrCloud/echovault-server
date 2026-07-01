package grpc

import (
	"context"
	"errors"

	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SongHandler struct {
	songpb.UnimplementedSongServiceServer
	svc *song.Service
}

func NewSongHandler(svc *song.Service) *SongHandler {
	return &SongHandler{svc: svc}
}

func (h *SongHandler) CheckSongsByHash(ctx context.Context, req *songpb.CheckSongsByHashRequest) (*songpb.CheckSongsByHashResponse, error) {
	results, err := h.svc.CheckSongsByHash(ctx, req.FileHashes)
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

func (h *SongHandler) PublishSong(ctx context.Context, req *songpb.PublishSongRequest) (*songpb.PublishSongResponse, error) {
	s, err := h.svc.PublishSong(ctx, req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &songpb.PublishSongResponse{Song: s}, nil
}

func (h *SongHandler) GetSong(ctx context.Context, req *songpb.GetSongRequest) (*songpb.GetSongResponse, error) {
	s, err := h.svc.GetSong(ctx, req.Id)
	if err != nil {
		code := codes.NotFound
		if errors.Is(err, errors.New("song not found")) {
			code = codes.NotFound
		}
		return nil, status.Error(code, err.Error())
	}
	return &songpb.GetSongResponse{Song: s}, nil
}

func (h *SongHandler) SearchSongs(ctx context.Context, req *songpb.SearchSongsRequest) (*songpb.SearchSongsResponse, error) {
	songs, err := h.svc.SearchSongs(ctx, req.Query, int(req.Pagination.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &songpb.SearchSongsResponse{Songs: songs}, nil
}

func (h *SongHandler) ListSongs(ctx context.Context, req *songpb.ListSongsRequest) (*songpb.ListSongsResponse, error) {
	limit := int(req.Pagination.GetPageSize())
	if limit <= 0 {
		limit = 20
	}
	songs, err := h.svc.ListSongs(ctx, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &songpb.ListSongsResponse{Songs: songs}, nil
}

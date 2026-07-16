package grpc

import (
	"context"

	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PlaylistHandler implements the PlaylistService gRPC server.
type PlaylistHandler struct {
	playlistpb.UnimplementedPlaylistServiceServer

	svc *playlist.Service
}

// NewPlaylistHandler creates a new PlaylistHandler.
func NewPlaylistHandler(svc *playlist.Service) *PlaylistHandler {
	return &PlaylistHandler{svc: svc}
}

// CreatePlaylist creates a new playlist for the authenticated user.
func (h *PlaylistHandler) CreatePlaylist(ctx context.Context, req *playlistpb.CreatePlaylistRequest) (*playlistpb.CreatePlaylistResponse, error) {
	userID := GetUserID(ctx)
	p, err := h.svc.CreatePlaylist(ctx, req.GetName(), req.GetDescription(), userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.CreatePlaylistResponse{Playlist: playlistEntToProto(p)}, nil
}

// GetPlaylist returns a playlist by ID.
func (h *PlaylistHandler) GetPlaylist(ctx context.Context, req *playlistpb.GetPlaylistRequest) (*playlistpb.GetPlaylistResponse, error) {
	p, err := h.svc.GetPlaylist(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.GetPlaylistResponse{Playlist: playlistEntToProto(p)}, nil
}

// ListPlaylists returns all playlists for the authenticated user.
func (h *PlaylistHandler) ListPlaylists(ctx context.Context, _ *playlistpb.ListPlaylistsRequest) (*playlistpb.ListPlaylistsResponse, error) {
	userID := GetUserID(ctx)
	playlists, err := h.svc.ListPlaylists(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	pb := make([]*playlistpb.Playlist, len(playlists))
	for i, p := range playlists {
		pb[i] = playlistEntToProto(p)
	}
	return &playlistpb.ListPlaylistsResponse{Playlists: pb}, nil
}

// AddSong adds a song to a playlist.
func (h *PlaylistHandler) AddSong(ctx context.Context, req *playlistpb.AddSongRequest) (*playlistpb.AddSongResponse, error) {
	userID := GetUserID(ctx)
	ps, err := h.svc.AddSong(ctx, req.GetPlaylistId(), req.GetSongId(), userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.AddSongResponse{PlaylistSong: &playlistpb.PlaylistSong{
		PlaylistId: ps.PlaylistID, SongId: ps.SongID, Position: ps.Position,
	}}, nil
}

// RemoveSong removes a song from a playlist.
func (h *PlaylistHandler) RemoveSong(ctx context.Context, req *playlistpb.RemoveSongRequest) (*playlistpb.RemoveSongResponse, error) {
	err := h.svc.RemoveSong(ctx, req.GetPlaylistId(), req.GetSongId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.RemoveSongResponse{}, nil
}

func playlistEntToProto(p *ent.Playlist) *playlistpb.Playlist {
	if p == nil {
		return nil
	}
	return &playlistpb.Playlist{
		Id: p.ID, Name: p.Name, Description: p.Description,
		OwnerId: p.OwnerID, SongCount: int32(p.SongCount), //nolint:gosec
		Type: playlistpb.Playlist_TYPE_USER,
	}
}

// UpdatePlaylist updates a playlist's name and description.
func (h *PlaylistHandler) UpdatePlaylist(ctx context.Context, req *playlistpb.UpdatePlaylistRequest) (*playlistpb.UpdatePlaylistResponse, error) {
	p, err := h.svc.UpdatePlaylist(ctx, req.GetId(), req.GetName(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.UpdatePlaylistResponse{Playlist: playlistEntToProto(p)}, nil
}

// DeletePlaylist deletes a playlist by ID.
func (h *PlaylistHandler) DeletePlaylist(ctx context.Context, req *playlistpb.DeletePlaylistRequest) (*playlistpb.DeletePlaylistResponse, error) {
	err := h.svc.DeletePlaylist(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.DeletePlaylistResponse{}, nil
}

// ReorderSongs reorders songs in a playlist.
func (h *PlaylistHandler) ReorderSongs(ctx context.Context, req *playlistpb.ReorderSongsRequest) (*playlistpb.ReorderSongsResponse, error) {
	err := h.svc.ReorderSongs(ctx, req.GetPlaylistId(), req.GetSongIds())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &playlistpb.ReorderSongsResponse{}, nil
}

// ListPlaylistSongs returns all songs in a playlist.
func (h *PlaylistHandler) ListPlaylistSongs(ctx context.Context, req *playlistpb.ListPlaylistSongsRequest) (*playlistpb.ListPlaylistSongsResponse, error) {
	songs, err := h.svc.ListPlaylistSongs(ctx, req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	pb := make([]*playlistpb.PlaylistSong, len(songs))
	for i, s := range songs {
		pb[i] = &playlistpb.PlaylistSong{
			PlaylistId: s.PlaylistID,
			SongId:     s.SongID,
			Position:   s.Position,
			AddedBy:    s.AddedBy,
		}
	}
	return &playlistpb.ListPlaylistSongsResponse{Songs: pb}, nil
}

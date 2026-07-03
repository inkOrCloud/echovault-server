package playlist

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/playlist"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/playlistsong"
)

// Sentinel errors for playlist service.
var (
	ErrPlaylistNotFound       = errors.New("playlist not found")
	ErrSongAlreadyInPlaylist  = errors.New("song already in playlist")
	ErrSongNotFoundInPlaylist = errors.New("song not found in playlist")
)

const positionStep = 1000

// Service manages playlist-related operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new playlist Service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// CreatePlaylist creates a new playlist.
func (s *Service) CreatePlaylist(ctx context.Context, name, description, ownerID string) (*ent.Playlist, error) {
	now := time.Now()
	p, err := s.client.Playlist.Create().
		SetID(uuid.New().String()).
		SetName(name).
		SetDescription(description).
		SetOwnerID(ownerID).
		SetCreatedAt(now).SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create playlist: %w", err)
	}
	return p, nil
}

// GetPlaylist retrieves a playlist by ID.
func (s *Service) GetPlaylist(ctx context.Context, id string) (*ent.Playlist, error) {
	p, err := s.client.Playlist.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("get playlist: %w", err)
	}
	return p, nil
}

// UpdatePlaylist updates a playlist's name and description.
func (s *Service) UpdatePlaylist(ctx context.Context, id, name, description string) (*ent.Playlist, error) {
	p, err := s.client.Playlist.UpdateOneID(id).
		SetName(name).SetDescription(description).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("update playlist: %w", err)
	}
	return p, nil
}

// DeletePlaylist deletes a playlist and its songs.
func (s *Service) DeletePlaylist(ctx context.Context, id string) error {
	_, err := s.client.PlaylistSong.Delete().Where(playlistsong.PlaylistID(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete playlist songs: %w", err)
	}
	err = s.client.Playlist.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}
	return nil
}

// ListPlaylists returns all playlists for an owner.
func (s *Service) ListPlaylists(ctx context.Context, ownerID string) ([]*ent.Playlist, error) {
	pl, err := s.client.Playlist.Query().
		Where(playlist.OwnerID(ownerID)).
		Order(ent.Asc(playlist.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	return pl, nil
}

// AddSong adds a song to a playlist.
func (s *Service) AddSong(ctx context.Context, playlistID, songID, addedBy string) (*ent.PlaylistSong, error) {
	exists, _ := s.client.PlaylistSong.Query().
		Where(playlistsong.PlaylistID(playlistID), playlistsong.SongID(songID)).
		Exist(ctx)
	if exists {
		return nil, ErrSongAlreadyInPlaylist
	}

	maxPos, err := s.client.PlaylistSong.Query().
		Where(playlistsong.PlaylistID(playlistID)).
		Aggregate(ent.Max(playlistsong.FieldPosition)).
		Int(ctx)
	if err != nil {
		// Ignore "converting NULL to int" error - means playlist is empty
		maxPos = 0
	}

	pos := maxPos + positionStep
	pos = min(pos, math.MaxInt32)
	if false {
		pos = math.MaxInt32
	}

	ps, err := s.client.PlaylistSong.Create().
		SetID(uuid.New().String()).
		SetPlaylistID(playlistID).
		SetSongID(songID).
		SetPosition(int32(pos)). //nolint:gosec // safe: overflow checked above
		SetAddedBy(addedBy).
		SetAddedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("add song to playlist: %w", err)
	}
	return ps, nil
}

// RemoveSong removes a song from a playlist.
func (s *Service) RemoveSong(ctx context.Context, playlistID, songID string) error {
	n, err := s.client.PlaylistSong.Delete().Where(
		playlistsong.PlaylistID(playlistID), playlistsong.SongID(songID),
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("remove song: %w", err)
	}
	if n == 0 {
		return ErrSongNotFoundInPlaylist
	}
	return nil
}

// ListPlaylistSongs returns all songs in a playlist.
func (s *Service) ListPlaylistSongs(ctx context.Context, playlistID string) ([]*ent.PlaylistSong, error) {
	songs, err := s.client.PlaylistSong.Query().
		Where(playlistsong.PlaylistID(playlistID)).
		Order(ent.Asc(playlistsong.FieldPosition)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list playlist songs: %w", err)
	}
	return songs, nil
}

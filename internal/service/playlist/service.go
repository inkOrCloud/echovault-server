package playlist

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/playlist"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/playlistsong"
)

type Service struct {
	client *ent.Client
}

func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

func (s *Service) CreatePlaylist(ctx context.Context, name, description, ownerID string) (*ent.Playlist, error) {
	now := time.Now()
	return s.client.Playlist.Create().
		SetID(uuid.New().String()).
		SetName(name).
		SetDescription(description).
		SetOwnerID(ownerID).
		SetCreatedAt(now).SetUpdatedAt(now).
		Save(ctx)
}

func (s *Service) GetPlaylist(ctx context.Context, id string) (*ent.Playlist, error) {
	p, err := s.client.Playlist.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) { return nil, errors.New("playlist not found") }
		return nil, err
	}
	return p, nil
}

func (s *Service) UpdatePlaylist(ctx context.Context, id, name, description string) (*ent.Playlist, error) {
	p, err := s.client.Playlist.UpdateOneID(id).
		SetName(name).SetDescription(description).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) { return nil, errors.New("playlist not found") }
		return nil, err
	}
	return p, nil
}

func (s *Service) DeletePlaylist(ctx context.Context, id string) error {
	// 删除歌单内所有歌曲
	s.client.PlaylistSong.Delete().Where(playlistsong.PlaylistID(id)).Exec(ctx)
	// 删除歌单
	return s.client.Playlist.DeleteOneID(id).Exec(ctx)
}

func (s *Service) ListPlaylists(ctx context.Context, ownerID string) ([]*ent.Playlist, error) {
	return s.client.Playlist.Query().
		Where(playlist.OwnerID(ownerID)).
		Order(ent.Asc(playlist.FieldCreatedAt)).
		All(ctx)
}

func (s *Service) AddSong(ctx context.Context, playlistID, songID, addedBy string) (*ent.PlaylistSong, error) {
	// 检查是否已存在
	exists, _ := s.client.PlaylistSong.Query().
		Where(playlistsong.PlaylistID(playlistID), playlistsong.SongID(songID)).
		Exist(ctx)
	if exists { return nil, errors.New("song already in playlist") }

	// 浮动排序：max(position) + 1000
	maxPos, _ := s.client.PlaylistSong.Query().
		Where(playlistsong.PlaylistID(playlistID)).
		Aggregate(ent.Max(playlistsong.FieldPosition)).
		Int(ctx)

	return s.client.PlaylistSong.Create().
		SetID(uuid.New().String()).
		SetPlaylistID(playlistID).
		SetSongID(songID).
		SetPosition(int32(maxPos + 1000)).
		SetAddedBy(addedBy).
		SetCreatedAt(time.Now()).
		Save(ctx)
}

func (s *Service) RemoveSong(ctx context.Context, playlistID, songID string) error {
	n, err := s.client.PlaylistSong.Delete().Where(
		playlistsong.PlaylistID(playlistID), playlistsong.SongID(songID),
	).Exec(ctx)
	if err != nil { return err }
	if n == 0 { return errors.New("song not found in playlist") }
	return nil
}

func (s *Service) ListPlaylistSongs(ctx context.Context, playlistID string) ([]*ent.PlaylistSong, error) {
	return s.client.PlaylistSong.Query().
		Where(playlistsong.PlaylistID(playlistID)).
		Order(ent.Asc(playlistsong.FieldPosition)).
		All(ctx)
}

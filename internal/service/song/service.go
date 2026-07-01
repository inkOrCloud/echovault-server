package song

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/song"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
)

type CheckResult struct {
	FileHash string
	Exists   bool
	Song     *songpb.Song
}

type Service struct {
	client *ent.Client
}

func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

func (s *Service) CheckSongsByHash(ctx context.Context, fileHashes []string) ([]*CheckResult, error) {
	records, err := s.client.Song.Query().
		Where(song.FileHashIn(fileHashes...), song.IsDeleted(false)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	hashMap := make(map[string]*ent.Song)
	for _, r := range records {
		hashMap[r.FileHash] = r
	}

	results := make([]*CheckResult, len(fileHashes))
	for i, h := range fileHashes {
		results[i] = &CheckResult{FileHash: h}
		if r, ok := hashMap[h]; ok {
			results[i].Exists = true
			results[i].Song = entToProto(r)
		}
	}
	return results, nil
}

func (s *Service) PublishSong(ctx context.Context, req *songpb.PublishSongRequest) (*songpb.Song, error) {
	now := time.Now()
	r, err := s.client.Song.Create().
		SetID(uuid.New().String()).
		SetTitle(req.Title).
		SetArtist(req.Artist).
		SetAlbum(req.Album).
		SetGenre(req.Genre).
		SetFileName(req.FileName).
		SetFileSize(req.FileSize).
		SetFileHash(req.FileHash).
		SetMimeType(req.MimeType).
		SetYear(req.Year).
		SetSource("uploaded").
		SetFileStatus("uploaded").
		SetVersion(1).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return entToProto(r), nil
}

func (s *Service) GetSong(ctx context.Context, id string) (*songpb.Song, error) {
	r, err := s.client.Song.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("song not found")
		}
		return nil, err
	}
	return entToProto(r), nil
}

func (s *Service) SearchSongs(ctx context.Context, query string, limit int) ([]*songpb.Song, error) {
	pattern := query
	records, err := s.client.Song.Query().
		Where(
			song.And(
				song.IsDeleted(false),
				song.Or(
					song.TitleContainsFold(pattern),
					song.ArtistContainsFold(pattern),
					song.AlbumContainsFold(pattern),
				),
			),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return entListToProto(records), nil
}

func (s *Service) ListSongs(ctx context.Context, limit, offset int) ([]*songpb.Song, error) {
	records, err := s.client.Song.Query().
		Where(song.IsDeleted(false)).
		Order(ent.Asc(song.FieldTitle)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return entListToProto(records), nil
}

func entToProto(r *ent.Song) *songpb.Song {
	return &songpb.Song{
		Id:          r.ID,
		Title:       r.Title,
		Artist:      r.Artist,
		Album:       r.Album,
		Genre:       r.Genre,
		TrackNumber: r.TrackNumber,
		DiscNumber:  r.DiscNumber,
		DurationMs:  r.DurationMs,
		Year:        r.Year,
		FileName:    r.FileName,
		FileSize:    r.FileSize,
		FileHash:    r.FileHash,
		MimeType:    r.MimeType,
		Bitrate:     r.Bitrate,
		SampleRate:  r.SampleRate,
		Version:     r.Version,
	}
}

func entListToProto(records []*ent.Song) []*songpb.Song {
	result := make([]*songpb.Song, len(records))
	for i, r := range records {
		result[i] = entToProto(r)
	}
	return result
}

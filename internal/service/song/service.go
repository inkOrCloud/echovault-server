package song

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/song"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
)

// ErrSongNotFound indicates the song was not found.
var ErrSongNotFound = errors.New("song not found")

// CheckResult describes the result of a hash check.
type CheckResult struct {
	FileHash string
	Exists   bool
	Song     *songpb.Song
}

// Service manages song-related operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new song Service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// CheckSongsByHash checks which of the given hashes already exist.
func (s *Service) CheckSongsByHash(ctx context.Context, fileHashes []string) ([]*CheckResult, error) {
	records, err := s.client.Song.Query().
		Where(song.FileHashIn(fileHashes...), song.IsDeleted(false)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query songs by hash: %w", err)
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

// PublishSong creates a new song record.
func (s *Service) PublishSong(ctx context.Context, req *songpb.PublishSongRequest) (*songpb.Song, error) {
	now := time.Now()
	r, err := s.client.Song.Create().
		SetID(uuid.New().String()).
		SetTitle(req.GetTitle()).
		SetArtist(req.GetArtist()).
		SetAlbum(req.GetAlbum()).
		SetGenre(req.GetGenre()).
		SetFileName(req.GetFileName()).
		SetFileSize(req.GetFileSize()).
		SetFileHash(req.GetFileHash()).
		SetMimeType(req.GetMimeType()).
		SetYear(req.GetYear()).
		SetSource("uploaded").
		SetFileStatus("uploaded").
		SetVersion(1).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("save song: %w", err)
	}
	return entToProto(r), nil
}

// GetSong retrieves a song by ID.
func (s *Service) GetSong(ctx context.Context, id string) (*songpb.Song, error) {
	r, err := s.client.Song.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrSongNotFound
		}
		return nil, fmt.Errorf("get song: %w", err)
	}
	return entToProto(r), nil
}

// SearchSongs searches songs by query string.
func (s *Service) SearchSongs(ctx context.Context, query string, limit int) ([]*songpb.Song, error) {
	records, err := s.client.Song.Query().
		Where(
			song.And(
				song.IsDeleted(false),
				song.Or(
					song.TitleContainsFold(query),
					song.ArtistContainsFold(query),
					song.AlbumContainsFold(query),
				),
			),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("search songs: %w", err)
	}
	return entListToProto(records), nil
}

// ListSongs returns a paginated list of songs.
func (s *Service) ListSongs(ctx context.Context, limit, offset int) ([]*songpb.Song, error) {
	records, err := s.client.Song.Query().
		Where(song.IsDeleted(false)).
		Order(ent.Asc(song.FieldTitle)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list songs: %w", err)
	}
	return entListToProto(records), nil
}

// UpdateFromScan populates empty fields from scanned metadata.
func (s *Service) UpdateFromScan(ctx context.Context, songID, title, artist, album, genre string,
	trackNumber, discNumber, year int32,
	fileHash, fileName, mimeType string, fileSize int64) error {

	r, err := s.client.Song.Get(ctx, songID)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrSongNotFound
		}
		return fmt.Errorf("get song: %w", err)
	}

	u := r.Update()
	applyIfEmpty := applyMetadataIfEmpty(u, r, title, artist, album, genre,
		trackNumber, discNumber, year, fileHash, fileName, mimeType, fileSize)

	if !applyIfEmpty {
		return nil
	}

	u.SetUpdatedAt(time.Now())
	if _, err := u.Save(ctx); err != nil {
		return fmt.Errorf("update song: %w", err)
	}
	return nil
}

// applyMetadataIfEmpty fills in empty fields from metadata. Returns true if any field was updated.
func applyMetadataIfEmpty(u *ent.SongUpdateOne, r *ent.Song,
	title, artist, album, genre string,
	trackNumber, discNumber, year int32,
	fileHash, fileName, mimeType string, fileSize int64) bool {

	updated := false

	if r.Title == "" && title != "" {
		u.SetTitle(title)
		updated = true
	}
	if r.Artist == "" && artist != "" {
		u.SetArtist(artist)
		updated = true
	}
	if r.Album == "" && album != "" {
		u.SetAlbum(album)
		updated = true
	}
	if r.Genre == "" && genre != "" {
		u.SetGenre(genre)
		updated = true
	}
	if r.TrackNumber == 0 && trackNumber > 0 {
		u.SetTrackNumber(trackNumber)
		updated = true
	}
	if r.DiscNumber == 0 && discNumber > 0 {
		u.SetDiscNumber(discNumber)
		updated = true
	}
	if r.Year == 0 && year > 0 {
		u.SetYear(year)
		updated = true
	}
	if r.FileSize == 0 && fileSize > 0 {
		u.SetFileSize(fileSize)
		updated = true
	}
	if r.FileHash == "" && fileHash != "" {
		u.SetFileHash(fileHash)
		updated = true
	}
	if r.MimeType == "" && mimeType != "" {
		u.SetMimeType(mimeType)
		updated = true
	}
	if r.FileName == "" && fileName != "" {
		u.SetFileName(fileName)
		updated = true
	}

	return updated
}

func entToProto(r *ent.Song) *songpb.Song {
	return &songpb.Song{
		Id:          r.ID,
		Title:       r.Title,
		Artist:      r.Artist,
		Album:       r.Album,
		Genre:       r.Genre,
		FileName:    r.FileName,
		FileSize:    r.FileSize,
		FileHash:    r.FileHash,
		MimeType:    r.MimeType,
		TrackNumber: r.TrackNumber,
		DiscNumber:  r.DiscNumber,
		Year:        r.Year,
	}
}

func entListToProto(records []*ent.Song) []*songpb.Song {
	songs := make([]*songpb.Song, len(records))
	for i, r := range records {
		songs[i] = entToProto(r)
	}
	return songs
}

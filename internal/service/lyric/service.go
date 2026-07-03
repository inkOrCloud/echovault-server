package lyric

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"google.golang.org/protobuf/types/known/timestamppb"

	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/lyric"
)

// ErrLyricNotFound indicates the lyric was not found.
var ErrLyricNotFound = errors.New("lyric not found")

// Service manages lyric-related operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new lyric Service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// SaveLyric creates or updates a lyric entry.
func (s *Service) SaveLyric(ctx context.Context, songID, content string, typ lyricpb.Lyric_Type, language string) (*lyricpb.Lyric, error) {
	typStr := typeToStr(typ)
	now := time.Now()

	existing, _ := s.client.Lyric.Query().
		Where(lyric.SongID(songID), lyric.Type(typStr), lyric.Language(language)).
		First(ctx)
	if existing != nil {
		r, err := s.client.Lyric.UpdateOne(existing).
			SetContent(content).SetUpdatedAt(now).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("update lyric: %w", err)
		}
		return entToProto(r), nil
	}

	r, err := s.client.Lyric.Create().
		SetID(uuid.New().String()).
		SetSongID(songID).SetContent(content).
		SetType(typStr).SetLanguage(language).
		SetUpdatedAt(now).SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create lyric: %w", err)
	}
	return entToProto(r), nil
}

// GetLyric retrieves a lyric by song ID, language, and type.
func (s *Service) GetLyric(ctx context.Context, songID, language string, typ lyricpb.Lyric_Type) (*lyricpb.Lyric, error) {
	query := s.client.Lyric.Query().Where(lyric.SongID(songID))
	if language != "" {
		query = query.Where(lyric.Language(language))
	}
	if typ != lyricpb.Lyric_TYPE_UNSPECIFIED {
		query = query.Where(lyric.Type(typeToStr(typ)))
	}
	r, err := query.First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrLyricNotFound
		}
		return nil, fmt.Errorf("get lyric: %w", err)
	}
	return entToProto(r), nil
}

// GetAllLyrics retrieves all lyrics for a song.
func (s *Service) GetAllLyrics(ctx context.Context, songID string) ([]*lyricpb.Lyric, error) {
	records, err := s.client.Lyric.Query().Where(lyric.SongID(songID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all lyrics: %w", err)
	}
	result := make([]*lyricpb.Lyric, len(records))
	for i, r := range records {
		result[i] = entToProto(r)
	}
	return result, nil
}

// DeleteLyric removes a lyric entry.
func (s *Service) DeleteLyric(ctx context.Context, songID string, typ lyricpb.Lyric_Type, language string) error {
	n, err := s.client.Lyric.Delete().Where(
		lyric.SongID(songID), lyric.Type(typeToStr(typ)), lyric.Language(language),
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete lyric: %w", err)
	}
	if n == 0 {
		return ErrLyricNotFound
	}
	return nil
}

func typeToStr(t lyricpb.Lyric_Type) string {
	switch t {
	case lyricpb.Lyric_TYPE_ORIGINAL:
		return "TYPE_ORIGINAL"
	case lyricpb.Lyric_TYPE_TRANSLATION:
		return "TYPE_TRANSLATION"
	case lyricpb.Lyric_TYPE_PHONETIC:
		return "TYPE_PHONETIC"
	default:
		return "TYPE_ORIGINAL"
	}
}

func entToProto(r *ent.Lyric) *lyricpb.Lyric {
	t := lyricpb.Lyric_TYPE_UNSPECIFIED
	switch r.Type {
	case "TYPE_ORIGINAL":
		t = lyricpb.Lyric_TYPE_ORIGINAL
	case "TYPE_TRANSLATION":
		t = lyricpb.Lyric_TYPE_TRANSLATION
	case "TYPE_PHONETIC":
		t = lyricpb.Lyric_TYPE_PHONETIC
	}
	s := lyricpb.Lyric_SOURCE_UNSPECIFIED
	switch r.Source {
	case "SOURCE_EMBEDDED":
		s = lyricpb.Lyric_SOURCE_EMBEDDED
	case "SOURCE_MANUAL":
		s = lyricpb.Lyric_SOURCE_MANUAL
	case "SOURCE_FETCHED":
		s = lyricpb.Lyric_SOURCE_FETCHED
	}
	return &lyricpb.Lyric{
		SongId:    r.SongID,
		Content:   r.Content,
		Type:      t,
		Language:  r.Language,
		OffsetMs:  r.OffsetMs,
		Source:    s,
		Version:   r.Version,
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

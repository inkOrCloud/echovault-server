package song

import (
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

func EntToProto(s *ent.Song) *songpb.Song {
	if s == nil {
		return nil
	}
	return &songpb.Song{
		Id:          s.ID,
		Title:       s.Title,
		Artist:      s.Artist,
		Album:       s.Album,
		Genre:       s.Genre,
		TrackNumber: s.TrackNumber,
		DiscNumber:  s.DiscNumber,
		DurationMs:  s.DurationMs,
		Year:        s.Year,
		FileName:    s.FileName,
		FileSize:    s.FileSize,
		FileHash:    s.FileHash,
		MimeType:    s.MimeType,
		Bitrate:     s.Bitrate,
		SampleRate:  s.SampleRate,
		OwnerId:     s.OwnerID,
		Version:     s.Version,
		IsDeleted:   s.IsDeleted,
		CreatedAt:   convert.PTime(s.CreatedAt),
		UpdatedAt:   convert.PTime(s.UpdatedAt),
	}
}

func EntsToProtoList(songs []*ent.Song) []*songpb.Song {
	result := make([]*songpb.Song, len(songs))
	for i, s := range songs {
		result[i] = EntToProto(s)
	}
	return result
}

package playlist

import (
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

func EntToProto(p *ent.Playlist) *playlistpb.Playlist {
	if p == nil {
		return nil
	}
	return &playlistpb.Playlist{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CoverUrl:    p.CoverURL,
		OwnerId:     p.OwnerID,
		IsPublic:    p.IsPublic,
		SongCount:   p.SongCount,
		Version:     p.Version,
		CreatedAt:   convert.PTime(p.CreatedAt),
		UpdatedAt:   convert.PTime(p.UpdatedAt),
	}
}

func playlistTypeToProto(s string) playlistpb.Playlist_Type {
	switch s {
	case "user":
		return playlistpb.Playlist_TYPE_USER
	case "favorite":
		return playlistpb.Playlist_TYPE_FAVORITE
	case "auto":
		return playlistpb.Playlist_TYPE_AUTO
	default:
		return playlistpb.Playlist_TYPE_UNSPECIFIED
	}
}

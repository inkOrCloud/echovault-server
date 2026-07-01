package rest

import "github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"

// SongUpdater 定义上传音频后补充元数据的接口。
// 由 adapter 在 main.go 中对接 SongService。
// 实现应查询 Song 记录，仅当字段值为空时才用 metadata 填充。
type SongUpdater interface {
	// UpdateFromScan 用音频文件提取的元数据补充 Song 记录中用户未填的字段。
	UpdateFromScan(songID string, meta *metadata.AudioMetadata, fileSize int64) error
}

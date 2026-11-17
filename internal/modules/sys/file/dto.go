package file

import (
	"io"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
)

// View 不暴露内部存储路径和摘要。
type View struct {
	ID           model.ID  `json:"id"`
	OriginalName string    `json:"originalName"`
	Size         int64     `json:"size"`
	Mime         string    `json:"mime"`
	DownloadURL  string    `json:"downloadUrl"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Download 持有经对象授权后打开的文件流。
type Download struct {
	View
	Body io.ReadCloser
}

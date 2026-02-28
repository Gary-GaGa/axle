package skill

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// ImageInfo holds metadata about an image file.
type ImageInfo struct {
	Width    int
	Height   int
	Format   string
	FileSize int64
	FileName string
}

// String returns a human-readable summary of the image.
func (i ImageInfo) String() string {
	return fmt.Sprintf(
		"🖼 *圖片資訊*\n\n"+
			"📐 尺寸：%d × %d\n"+
			"🎨 格式：%s\n"+
			"📦 大小：%s\n"+
			"📄 檔名：`%s`",
		i.Width, i.Height,
		strings.ToUpper(i.Format),
		humanSize(i.FileSize),
		i.FileName,
	)
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// AnalyzeImage reads an image file and returns its metadata.
func AnalyzeImage(filePath string) (*ImageInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("無法開啟圖片: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("無法讀取檔案資訊: %w", err)
	}

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, fmt.Errorf("無法解析圖片格式: %w", err)
	}

	return &ImageInfo{
		Width:    cfg.Width,
		Height:   cfg.Height,
		Format:   format,
		FileSize: stat.Size(),
		FileName: filepath.Base(filePath),
	}, nil
}

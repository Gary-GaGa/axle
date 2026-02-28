package skill

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

const maxPDFTextSize = 30 * 1024 // 30KB, same as ReadCode limit

// ExtractPDFText extracts plain text from a PDF file.
// Returns the extracted text, truncated to maxPDFTextSize if needed.
func ExtractPDFText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("無法開啟 PDF: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		if buf.Len() > maxPDFTextSize {
			break
		}
	}

	result := buf.String()
	if len(result) > maxPDFTextSize {
		result = result[:maxPDFTextSize] + "\n\n⚠️ 文字過長，已截斷至 30KB"
	}
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("PDF 無法提取文字（可能為掃描圖片型 PDF）")
	}
	return fmt.Sprintf("📄 PDF 文字內容（%d 頁）：\n\n%s", r.NumPage(), result), nil
}

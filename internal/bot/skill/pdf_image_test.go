package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPDFText_InvalidFile(t *testing.T) {
	_, err := ExtractPDFText("/nonexistent/file.pdf")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestExtractPDFText_NotAPDF(t *testing.T) {
	// Create a temp file that isn't a PDF
	tmp, err := os.CreateTemp("", "axle-test-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("this is not a PDF file")
	tmp.Close()

	_, err = ExtractPDFText(tmp.Name())
	if err == nil {
		t.Fatal("expected error for non-PDF file")
	}
}

func TestAnalyzeImage_InvalidFile(t *testing.T) {
	_, err := AnalyzeImage("/nonexistent/image.png")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestAnalyzeImage_NotAnImage(t *testing.T) {
	tmp, err := os.CreateTemp("", "axle-test-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("not an image")
	tmp.Close()

	_, err = AnalyzeImage(tmp.Name())
	if err == nil {
		t.Fatal("expected error for non-image file")
	}
}

func TestAnalyzeImage_ValidJPEG(t *testing.T) {
	// Create a minimal valid JPEG (smallest valid JPEG: SOI + EOI)
	// This won't decode properly via image.DecodeConfig, so let's create a tiny PNG instead
	// Minimal 1x1 red PNG
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // 8-bit RGB
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	tmp := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(tmp, pngData, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := AnalyzeImage(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Width != 1 || info.Height != 1 {
		t.Errorf("expected 1x1, got %dx%d", info.Width, info.Height)
	}
	if info.Format != "png" {
		t.Errorf("expected png, got %s", info.Format)
	}
	if info.String() == "" {
		t.Error("String() should return non-empty")
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{2621440, "2.5 MB"},
	}
	for _, tt := range tests {
		got := humanSize(tt.input)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestImageInfoString(t *testing.T) {
	info := ImageInfo{
		Width: 800, Height: 600,
		Format: "jpeg", FileSize: 12345,
		FileName: "photo.jpg",
	}
	s := info.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

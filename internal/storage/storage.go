package storage

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type StorageProvider interface {
	Save(filename string, r io.Reader) (storedPath string, size int64, mimeType string, err error)
	Open(storedPath string) (io.ReadCloser, error)
	Delete(storedPath string) error
	GetURL(storedPath string) string
}

type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}
	return &LocalStorage{
		baseDir: baseDir,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

func (s *LocalStorage) Save(filename string, r io.Reader) (string, int64, string, error) {
	// Sanitize original filename and generate unique storage filename
	ext := filepath.Ext(filename)
	safeExt := strings.ToLower(strings.TrimPrefix(ext, "."))
	uniqueID := uuid.New().String()
	storedName := fmt.Sprintf("%s.%s", uniqueID, safeExt)
	destPath := filepath.Join(s.baseDir, storedName)

	// Ensure destination remains within base directory
	rel, err := filepath.Rel(s.baseDir, destPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", 0, "", fmt.Errorf("invalid storage path")
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer out.Close()

	// Sniff MIME type from first 512 bytes
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", 0, "", fmt.Errorf("failed to read file header: %w", err)
	}

	mimeType := http.DetectContentType(buf[:n])

	if n > 0 {
		if _, writeErr := out.Write(buf[:n]); writeErr != nil {
			return "", 0, "", fmt.Errorf("failed to write buffer: %w", writeErr)
		}
	}

	copied, copyErr := io.Copy(out, r)
	if copyErr != nil {
		return "", 0, "", fmt.Errorf("failed to copy file body: %w", copyErr)
	}

	totalSize := int64(n) + copied
	return storedName, totalSize, mimeType, nil
}

func (s *LocalStorage) Open(storedPath string) (io.ReadCloser, error) {
	cleanName := filepath.Base(storedPath)
	fullPath := filepath.Join(s.baseDir, cleanName)

	rel, err := filepath.Rel(s.baseDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("invalid file path")
	}

	return os.Open(fullPath)
}

func (s *LocalStorage) Delete(storedPath string) error {
	cleanName := filepath.Base(storedPath)
	fullPath := filepath.Join(s.baseDir, cleanName)
	return os.Remove(fullPath)
}

func (s *LocalStorage) GetURL(storedPath string) string {
	cleanName := filepath.Base(storedPath)
	return fmt.Sprintf("%s/api/v1/documents/files/%s", s.baseURL, cleanName)
}

package storage

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	Root string
}

func New(root string) *Store {
	return &Store{Root: strings.TrimRight(root, "/")}
}

func (s *Store) EnsureDirs() error {
	if err := os.MkdirAll(filepath.Join(s.Root, "tmp"), 0o755); err != nil {
		return err
	}
	return nil
}

func (s *Store) ComputeMD5AndWriteTemp(file multipart.File) (string, uint64, string, error) {
	tmpDir := filepath.Join(s.Root, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", 0, "", err
	}
	tmp, err := os.CreateTemp(tmpDir, "upload-*")
	if err != nil {
		return "", 0, "", err
	}
	defer tmp.Close()

	h := md5.New()
	writer := io.MultiWriter(tmp, h)
	size, err := io.Copy(writer, file)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", 0, "", err
	}
	md5Hex := hex.EncodeToString(h.Sum(nil))
	return md5Hex, uint64(size), tmp.Name(), nil
}

func (s *Store) FinalPath(tenantID uint64, md5Hex string) string {
	prefix := md5Hex
	if len(md5Hex) >= 2 {
		prefix = md5Hex[:2]
	}
	return filepath.Join(fmt.Sprintf("tenant_%d", tenantID), prefix, md5Hex+".bin")
}

func (s *Store) MoveToFinal(tempPath string, finalRelPath string) error {
	abs, err := s.AbsPath(finalRelPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(abs); err == nil {
		_ = os.Remove(tempPath)
		return nil
	}
	return os.Rename(tempPath, abs)
}

func (s *Store) OpenForRead(relPath string) (*os.File, error) {
	abs, err := s.AbsPath(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
}

func (s *Store) Delete(relPath string) error {
	abs, err := s.AbsPath(relPath)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}

func (s *Store) AbsPath(relPath string) (string, error) {
	clean := filepath.Clean(relPath)
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return "", fmt.Errorf("invalid file path")
	}
	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(rootAbs, clean)
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid file path")
	}
	return abs, nil
}

func (s *Store) CleanupEmptyParents(relPath string) {
	abs, err := s.AbsPath(relPath)
	if err != nil {
		return
	}
	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return
	}
	dir := filepath.Dir(abs)
	for dir != rootAbs && dir != filepath.Dir(rootAbs) {
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

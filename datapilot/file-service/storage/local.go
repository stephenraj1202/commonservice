package storage

import (
	"io"
	"os"
	"path/filepath"
)

// Storage abstracts filesystem operations for uploaded files.
type Storage interface {
	// Save writes the content from r to the given filename, returning bytes written.
	Save(filename string, r io.Reader) (int64, error)
	// Open returns a ReadCloser for the given filename.
	Open(filename string) (io.ReadCloser, error)
	// Delete removes the given filename from storage.
	Delete(filename string) error
}

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	BasePath string
}

// Save creates BasePath/filename (creating directories as needed) and copies r into it.
func (s *LocalStorage) Save(filename string, r io.Reader) (int64, error) {
	dest := filepath.Join(s.BasePath, filename)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return 0, err
	}
	f, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

// Open returns a ReadCloser for BasePath/filename.
func (s *LocalStorage) Open(filename string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.BasePath, filename))
}

// Delete removes BasePath/filename from the filesystem.
func (s *LocalStorage) Delete(filename string) error {
	return os.Remove(filepath.Join(s.BasePath, filename))
}

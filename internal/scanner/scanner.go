package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/babyfaceeasy/repo-scanner/internal/model"
	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
)

// Scanner handles file scanning
type Scanner struct {
	logger logger.Logger
}

// New creates a new Scanner
func New(logger logger.Logger) *Scanner {
	return &Scanner{
		logger: logger,
	}
}

// Scan traverses the directory and finds files larger than the threshold
func (s *Scanner) Scan(root string, sizeThreshold int64) (*model.Output, error) {
	var files []model.FileInfo
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // continue
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // continue
		}

		if info.Size() > sizeThreshold {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("getting relative path for %s: %w", path, err)
			}
			files = append(files, model.FileInfo{
				Name: relPath,
				Size: info.Size(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning directory: %w", err)
	}

	return &model.Output{
		Total: len(files),
		Files: files,
	}, nil
}

package service

import (
	"os"

	"github.com/babyfaceeasy/repo-scanner/internal/config"
	"github.com/babyfaceeasy/repo-scanner/internal/github"
	"github.com/babyfaceeasy/repo-scanner/internal/output"
	"github.com/babyfaceeasy/repo-scanner/internal/scanner"
	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
)

// Service orchestrates the application logic
type Service struct {
	config  *config.ConfigParser
	github  github.GitHubClient
	scanner *scanner.Scanner
	output  *output.Writer
	logger  logger.Logger
}

// New creates a new Service
func New(cfg *config.ConfigParser, gh github.GitHubClient, sc *scanner.Scanner, out *output.Writer, logger logger.Logger) *Service {
	return &Service{
		config:  cfg,
		github:  gh,
		scanner: sc,
		output:  out,
		logger:  logger,
	}
}

// Scan executes the repository scanning process
func (s *Service) Scan(jsonStr string) error {
	cfg, err := s.config.Parse(jsonStr)
	if err != nil {
		return err
	}
	s.logger.Info("Config parsed", "clone_url", cfg.CloneURL, "size_mb", cfg.Size)

	cloneDir, err := os.MkdirTemp("", "repo-scan-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(cloneDir)
	s.logger.Info("Created temp dir", "path", cloneDir)

	if err := s.github.DownloadRepo(cfg.CloneURL, cloneDir); err != nil {
		return err
	}
	s.logger.Info("Repository downloaded", "path", cloneDir)

	sizeThreshold := int64(cfg.Size * 1024 * 1024)
	result, err := s.scanner.Scan(cloneDir, sizeThreshold)
	if err != nil {
		return err
	}
	s.logger.Info("File scan completed", "total_files", result.Total)

	return s.output.Write(result)
}

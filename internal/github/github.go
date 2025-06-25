package github

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
)

// GitHubClient defines the interface for GitHub interactions
type GitHubClient interface {
	DownloadRepo(cloneURL, destDir string) error
}

// Client is a GitHub API client
type Client struct {
	httpClient         *http.Client
	token              string
	logger             logger.Logger
	cloneURLToTarballURL func(string) (string, error)
}

// NewClient creates a new GitHub client
func NewClient(token string, logger logger.Logger) *Client {
	return &Client{
		httpClient:         &http.Client{},
		token:              token,
		logger:             logger,
		cloneURLToTarballURL: cloneURLToTarballURL,
	}
}

// DownloadRepo downloads the repository tarball and extracts it to destDir
func (c *Client) DownloadRepo(cloneURL, destDir string) error {
	tarballURL, err := c.cloneURLToTarballURL(cloneURL)
	if err != nil {
		return fmt.Errorf("converting clone URL: %w", err)
	}
	c.logger.Info("Converted clone URL", "clone_url", cloneURL, "tarball_url", tarballURL)

	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	c.logger.Info("Fetched tarball", "url", tarballURL)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tarball: %w", err)
		}

		parts := strings.SplitN(header.Name, "/", 2)
		var relPath string
		if len(parts) > 1 {
			relPath = parts[1]
		} else {
			continue
		}

		targetPath := filepath.Join(destDir, relPath)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", targetPath, err)
			}
			c.logger.Debug("Created directory", "path", targetPath)
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", targetPath, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("writing file %s: %w", targetPath, err)
			}
			outFile.Close()
			c.logger.Debug("Extracted file", "path", targetPath)
		}
	}

	c.logger.Info("Repository extracted", "dest_dir", destDir)
	return nil
}

// cloneURLToTarballURL converts a GitHub clone URL to a tarball URL
var cloneURLToTarballURL = func(cloneURL string) (string, error) {
	if !strings.HasPrefix(cloneURL, "https://github.com/") {
		return "", fmt.Errorf("invalid GitHub clone URL")
	}

	path := strings.TrimPrefix(cloneURL, "https://github.com/")
	path = strings.TrimSuffix(path, ".git")
	if path == "" {
		return "", fmt.Errorf("invalid repository path")
	}

	return fmt.Sprintf("https://api.github.com/repos/%s/tarball", path), nil
}
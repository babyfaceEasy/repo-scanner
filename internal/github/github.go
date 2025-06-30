package github

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
)

// GitHubClient defines the interface for GitHub interactions
type GitHubClient interface {
	DownloadRepo(cloneURL, destDir string) error
}

// Client is a GitHub API client
type Client struct {
	httpClient           *http.Client
	token                string
	logger               logger.Logger
	cloneURLToTarballURL func(string) (string, error)
}

// NewClient creates a new GitHub client
func NewClient(token string, logger logger.Logger) *Client {
	return &Client{
		httpClient:           &http.Client{},
		token:                token,
		logger:               logger,
		cloneURLToTarballURL: cloneURLToTarballURL,
	}
}

// DownloadRepoSequentially downloads the repository tarball and extracts it to destDir
func (c *Client) DownloadRepoSequentially(cloneURL, destDir string) error {
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

		// handle rate-limit reached
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 5 * time.Second // default fallback
			if val := resp.Header.Get("Retry-After"); val != "" {
				if secs, err := strconv.Atoi(val); err == nil {
					retryAfter = time.Duration(secs) * time.Second
				}
			}
			return &RetryAfterError{
				Err:        fmt.Errorf("rate limited: 429 Too Many Requests"),
				RetryAfter: retryAfter,
			}
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	c.logger.Info("Fetched tarball", "url", tarballURL)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
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
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
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

// DownloadRepo downloads the repository tarball and extracts it to destDir. does it using worker pattern
func (c *Client) DownloadRepo(cloneURL, destDir string) error {
	tarball, err := c.getTarballStream(cloneURL)
	if err != nil {
		return err
	}
	defer tarball.Close()

	c.logger.Debug("tarball stream got successfully")

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	c.logger.Debug("about to call extract tarball")

	return extractTarballConcurrently(tarball, destDir, c.logger)
}

func (c *Client) getTarballStream(cloneURL string) (io.ReadCloser, error) {
	tarballURL, err := cloneURLToTarballURL(cloneURL)
	if err != nil {
		return nil, fmt.Errorf("converting clone URL: %w", err)
	}
	c.logger.Info("Converted clone URL", "clone_url", cloneURL, "tarball_url", tarballURL)

	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching tarball: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 3 * time.Second // default
			if val := resp.Header.Get("Retry-After"); val != "" {
				if secs, err := strconv.Atoi(val); err == nil {
					retryAfter = time.Duration(secs) * time.Second
				}
			}
			return nil, &RetryAfterError{
				Err:        fmt.Errorf("rate limited: 429 Too Many Requests"),
				RetryAfter: retryAfter,
			}
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}

	return struct {
		io.Reader
		io.Closer
	}{
		Reader: gzr,
		Closer: closer{gzr: gzr, body: resp.Body},
	}, nil
} // end of getTarballStream

type closer struct {
	gzr  *gzip.Reader
	body io.Closer
}

func (c closer) Close() error {
	gzrErr := c.gzr.Close()
	bodyErr := c.body.Close()
	if gzrErr != nil {
		return gzrErr
	}
	return bodyErr
}

type extractTask struct {
	data []byte
	path string
}

func extractTarballConcurrently(r io.Reader, destDir string, log logger.Logger) error {
    const workerCount = 4

    tr := tar.NewReader(r)
    tasks := make(chan extractTask, 20)
    errChan := make(chan error, workerCount)
    var wg sync.WaitGroup

    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            for task := range tasks {
                if err := writeFile(task.data, task.path); err != nil {
                    log.Error("Worker failed to write file", "worker", workerID, "path", task.path, "error", err)
                    errChan <- fmt.Errorf("worker %d: %w", workerID, err)
                    return
                }
                log.Debug("Extracted file", "worker", workerID, "path", task.path)
            }
        }(i)
    }

    for {
        header, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Error("Failed to read tarball", "error", err)
            close(tasks)
            wg.Wait()
            return fmt.Errorf("reading tarball: %w", err)
        }

        // log the header name for debugging
        log.Debug("Processing tar entry", "name", header.Name, "type", header.Typeflag)

        parts := strings.SplitN(header.Name, "/", 2)
        var relPath string
        if len(parts) < 2 {
            log.Debug("Skipping entry with no relative path", "name", header.Name)
            continue // skip entries like pax_global_header or root directory
        }
        relPath = parts[1]
        if relPath == "" {
            log.Debug("Skipping empty relative path", "name", header.Name)
            continue // skip root directory entries with empty relPath
        }

        targetPath := filepath.Join(destDir, relPath)

        // Path traversal check
        cleanedTarget := filepath.Clean(targetPath)
        cleanedDestDir := filepath.Clean(destDir)
        expectedPrefix := cleanedDestDir + string(os.PathSeparator)
        if !strings.HasPrefix(cleanedTarget, expectedPrefix) {
            log.Error("Path traversal detected", "header_name", header.Name, "rel_path", relPath, "target_path", targetPath, "expected_prefix", expectedPrefix)
            close(tasks)
            wg.Wait()
            return fmt.Errorf("illegal file path: %s (header: %s)", targetPath, header.Name)
        }

        switch header.Typeflag {
        case tar.TypeDir:
            if err := os.MkdirAll(targetPath, 0755); err != nil {
                log.Error("Failed to create directory", "path", targetPath, "error", err)
                close(tasks)
                wg.Wait()
                return fmt.Errorf("creating directory: %w", err)
            }
            log.Debug("Created directory", "path", targetPath)

        case tar.TypeReg:
            var buf bytes.Buffer
            if _, err := io.Copy(&buf, tr); err != nil {
                log.Error("Failed to read file from tar", "path", targetPath, "error", err)
                close(tasks)
                wg.Wait()
                return fmt.Errorf("reading file %s: %w", targetPath, err)
            }
            tasks <- extractTask{
                data: buf.Bytes(),
                path: targetPath,
            }
        default:
            log.Debug("Skipping unsupported tar entry type", "name", header.Name, "type", header.Typeflag)
        }
    }

    close(tasks)
    wg.Wait()
    close(errChan)

    if len(errChan) > 0 {
        err := <-errChan
        log.Error("Extraction failed", "error", err)
        return err
    }

    log.Info("Repository extracted", "dest_dir", destDir)
    return nil
}

var dirMutex sync.Mutex

func writeFile(data []byte, path string) error {
	dirMutex.Lock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		dirMutex.Unlock()
		return fmt.Errorf("creating parent dir: %w", err)
	}
	dirMutex.Unlock()

	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := outFile.Write(data); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

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

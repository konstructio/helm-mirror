package service

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/repo"
)

const (
	downloadedFileName = "downloaded-index.yaml"
	indexFileName      = "index.yaml"
	dirSeparator       = "/"
)

// GetServiceInterface defines a Get service
type GetServiceInterface interface {
	Get() error
}

// GetService structure definition
type GetService struct {
	config       repo.Entry
	verbose      bool
	ignoreErrors bool
	logger       *log.Logger
	newRootURL   string
	allVersions  bool
	chartName    string
	chartVersion string
}

// NewGetService return a new instace of GetService
func NewGetService(config repo.Entry, allVersions bool, verbose bool, ignoreErrors bool, logger *log.Logger, newRootURL string, chartName string, chartVersion string) *GetService {
	return &GetService{
		config:       config,
		verbose:      verbose,
		ignoreErrors: ignoreErrors,
		logger:       logger,
		newRootURL:   newRootURL,
		allVersions:  allVersions,
		chartName:    chartName,
		chartVersion: chartVersion,
	}
}

func (g *GetService) logVerbose(format string, args ...any) {
	if g.verbose {
		g.logger.Printf(format, args...)
	}
}

// Get methods downloads the index file and the Helm charts to the working directory.
func (g *GetService) Get() error {
	chartRepo, err := repo.NewChartRepository(&g.config, getter.All(environment.EnvSettings{}))
	if err != nil {
		return fmt.Errorf("cannot construct chart repository: %w", err)
	}

	g.logVerbose("Downloading index file from %s", g.config.URL)
	downloadedIndexPath := path.Join(g.config.Name, downloadedFileName)
	if err := chartRepo.DownloadIndexFile(downloadedIndexPath); err != nil {
		return fmt.Errorf("cannot download index file: %w", err)
	}

	g.logVerbose("Loading local directory %q as repository", g.config.Name)
	if err := chartRepo.Load(); err != nil {
		return fmt.Errorf("cannot load index file: %w", err)
	}

	g.logVerbose("Creating a new local index and adding charts to it")
	index := search.NewIndex()
	index.AddRepo(chartRepo.Config.Name, chartRepo.IndexFile, (g.allVersions || g.chartVersion != ""))

	rexp := fmt.Sprintf("^.*%s.*", g.chartName)
	g.logVerbose("Searching for regexp %q in index file", rexp)

	results, err := index.Search(rexp, 1, true)
	if err != nil {
		return fmt.Errorf("cannot search index file: %w", err)
	}

	g.logVerbose("Found %d results from searching %q", len(results), rexp)

	for _, result := range results {
		g.logVerbose("Processing chart %q (version %s)", result.Chart.Name, result.Chart.Version)

		if g.chartName != "" && result.Chart.Name != g.chartName {
			continue
		}

		if g.chartVersion != "" && result.Chart.Version != g.chartVersion {
			continue
		}

		for _, val := range result.Chart.URLs {
			g.logVerbose("Found chart URL %q for chart %q (version %s)", val, result.Chart.Name, result.Chart.Version)

			chartURL, err := url.Parse(val)
			if err != nil {
				return fmt.Errorf("invalid chart URL %q: %w", val, err)
			}

			if chartURL.Scheme == "" {
				val = strings.TrimRight(g.config.URL, dirSeparator) + dirSeparator + val
			}

			g.logVerbose("Downloading chart %q (version %s) from %q", result.Chart.Name, result.Chart.Version, val)

			buf, err := chartRepo.Client.Get(val)
			if err != nil {
				if g.ignoreErrors {
					g.logger.Printf("WARNING: processing chart %s(%s) - %s", result.Name, result.Chart.Version, err)
					continue
				}
				return fmt.Errorf("cannot download chart %s(%s): %w", result.Name, result.Chart.Version, err)
			}

			chartFileName := fmt.Sprintf("%s-%s.tgz", result.Chart.Name, result.Chart.Version)
			chartPath := path.Join(g.config.Name, chartFileName)

			g.logVerbose("Writing chart %q (version %s) to %q", result.Chart.Name, result.Chart.Version, chartPath)
			if err := g.writeFile(chartPath, buf.Bytes()); err != nil {
				return fmt.Errorf("cannot write chart %s(%s): %w", result.Name, result.Chart.Version, err)
			}
		}
	}

	g.logVerbose("Preparing index file %q: rewriting URL: %q->%q", g.config.Name, g.config.URL, g.newRootURL)
	if err := g.prepareIndexFile(g.config.Name, g.config.URL, g.newRootURL); err != nil {
		return fmt.Errorf("cannot prepare index file: %w", err)
	}

	g.logVerbose("Operation completed successfully")
	return nil
}

func (g *GetService) writeFile(name string, content []byte) error {
	if err := os.WriteFile(name, content, 0o600); err != nil {
		if g.ignoreErrors {
			g.logger.Printf("Skipping due to ignore errors: Cannot write file %q (%d bytes): %s", name, len(content), err)
		} else {
			return fmt.Errorf("cannot write file %q: %w", name, err)
		}
	}

	return nil
}

func (g *GetService) prepareIndexFile(folder string, repoURL string, newRootURL string) error {
	downloadedPath := path.Join(folder, downloadedFileName)
	indexPath := path.Join(folder, indexFileName)

	if newRootURL != "" {
		indexContent, err := os.ReadFile(downloadedPath)
		if err != nil {
			return fmt.Errorf("cannot read index file: %w", err)
		}

		content := bytes.ReplaceAll(indexContent, []byte(repoURL), []byte(newRootURL))
		if err := g.writeFile(downloadedPath, content); err != nil {
			//nolint:nilerr // ignore error
			return nil
		}
	}

	if err := os.Rename(downloadedPath, indexPath); err != nil {
		return fmt.Errorf("cannot rename index file: %w", err)
	}
	return nil
}

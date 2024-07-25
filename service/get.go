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

// Get methods downloads the index file and the Helm charts to the working directory.
func (g *GetService) Get() error {
	chartRepo, err := repo.NewChartRepository(&g.config, getter.All(environment.EnvSettings{}))
	if err != nil {
		return fmt.Errorf("cannot construct chart repository: %w", err)
	}

	downloadedIndexPath := path.Join(g.config.Name, downloadedFileName)
	err = chartRepo.DownloadIndexFile(downloadedIndexPath)
	if err != nil {
		return fmt.Errorf("cannot download index file: %w", err)
	}

	err = chartRepo.Load()
	if err != nil {
		return fmt.Errorf("cannot load index file: %w", err)
	}

	index := search.NewIndex()
	index.AddRepo(chartRepo.Config.Name, chartRepo.IndexFile, (g.allVersions || g.chartVersion != ""))
	rexp := fmt.Sprintf("^.*%s.*", g.chartName)
	results, err := index.Search(rexp, 1, true)
	if err != nil {
		return fmt.Errorf("cannot search index file: %w", err)
	}

	for _, result := range results {
		if g.chartName != "" && result.Chart.Name != g.chartName {
			continue
		}
		if g.chartVersion != "" && result.Chart.Version != g.chartVersion {
			continue
		}
		for _, u := range result.Chart.URLs {
			chartURL, err := url.Parse(u)
			if err != nil {
				return fmt.Errorf("invalid chart URL %q: %w", u, err)
			}

			if chartURL.Scheme == "" {
				u = strings.TrimRight(g.config.URL, dirSeparator) + dirSeparator + u
			}

			buf, err := chartRepo.Client.Get(u)
			if err != nil {
				if g.ignoreErrors {
					g.logger.Printf("WARNING: processing chart %s(%s) - %s", result.Name, result.Chart.Version, err)
					continue
				}
				return fmt.Errorf("cannot download chart %s(%s): %w", result.Name, result.Chart.Version, err)
			}
			chartFileName := fmt.Sprintf("%s-%s.tgz", result.Chart.Name, result.Chart.Version)
			chartPath := path.Join(g.config.Name, chartFileName)

			if err := writeFile(chartPath, buf.Bytes(), g.logger, g.ignoreErrors); err != nil {
				return fmt.Errorf("cannot write chart %s(%s): %w", result.Name, result.Chart.Version, err)
			}
		}
	}

	err = prepareIndexFile(g.config.Name, g.config.URL, g.newRootURL, g.logger, g.ignoreErrors)
	if err != nil {
		return err
	}
	return nil
}

func writeFile(name string, content []byte, log *log.Logger, ignoreErrors bool) error {
	err := os.WriteFile(name, content, 0o600)
	if err != nil {
		if ignoreErrors {
			log.Printf("cannot write file %q: %s", name, err)
		} else {
			return fmt.Errorf("cannot write file %q: %w", name, err)
		}
	}
	return nil
}

func prepareIndexFile(folder string, repoURL string, newRootURL string, log *log.Logger, ignoreErrors bool) error {
	downloadedPath := path.Join(folder, downloadedFileName)
	indexPath := path.Join(folder, indexFileName)
	if newRootURL != "" {
		indexContent, err := os.ReadFile(downloadedPath)
		if err != nil {
			return fmt.Errorf("cannot read index file: %w", err)
		}
		content := bytes.ReplaceAll(indexContent, []byte(repoURL), []byte(newRootURL))
		if err := writeFile(downloadedPath, content, log, ignoreErrors); err != nil {
			//nolint:nilerr // ignore error
			return nil
		}
	}

	if err := os.Rename(downloadedPath, indexPath); err != nil {
		return fmt.Errorf("cannot rename index file: %w", err)
	}
	return nil
}

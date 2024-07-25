package service

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/konstructio/helm-mirror/formatter"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	tversion "k8s.io/helm/pkg/version"
)

// ImagesServiceInterface defines a Get service
type ImagesServiceInterface interface {
	Images() error
}

// ImagesService structure definition
type ImagesService struct {
	target         string
	formatter      formatter.Formatter
	verbose        bool
	ignoreErrors   bool
	exitWithErrors bool
	logger         *log.Logger
	buffer         bytes.Buffer
}

// NewImagesService return a new instace of ImagesService
func NewImagesService(target string, verbose bool, ignoreErrors bool, formatter formatter.Formatter, logger *log.Logger) *ImagesService {
	return &ImagesService{
		target:       target,
		formatter:    formatter,
		logger:       logger,
		verbose:      verbose,
		ignoreErrors: ignoreErrors,
	}
}

// Images extracts al the images in the Helm Charts downloaded by the get command
func (i *ImagesService) Images() error {
	//nolint:varnamelen
	fi, err := os.Stat(i.target)
	if err != nil {
		i.logger.Printf("error: cannot read target: %s", i.target)
		return fmt.Errorf("cannot stat target %q: %w", i.target, err)
	}

	if fi.IsDir() {
		err = i.processDirectory(i.target)
	} else {
		err = i.processTarget(i.target)
	}
	if err != nil {
		i.logger.Printf("error: procesing target %s: %s", i.target, err)
		return fmt.Errorf("cannot process target %q: %w", i.target, err)
	}

	if err := i.formatter.Output(i.buffer); err != nil {
		i.logger.Printf("writing output: %s", err)
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

func (i *ImagesService) processDirectory(target string) error {
	hasTgzCharts := false

	//nolint:varnamelen
	fi, err := os.Stat(i.target)
	if err != nil {
		i.logger.Printf("error: cannot read target: %s", i.target)
		return fmt.Errorf("cannot stat target %q: %w", i.target, err)
	}

	if !fi.IsDir() {
		return errors.New("error: inspectImages: processDirectory: target not a directory")
	}

	perr := i.processTarget(target)
	if perr != nil {
		err := filepath.Walk(target, func(dir string, info os.FileInfo, err error) error {
			if err != nil {
				i.logger.Printf("error: cannot access a dir %q: %v\n", dir, err)
				return err
			}
			if !info.IsDir() && strings.Contains(info.Name(), ".tgz") {
				hasTgzCharts = true
				err := i.processTarget(path.Join(target, info.Name()))
				if err != nil && i.ignoreErrors {
					i.exitWithErrors = true
				} else if err != nil {
					i.logger.Printf("error: cannot load chart: %s", err)
					return err
				}
			}
			return nil
		})
		if err != nil {
			i.logger.Printf("error walking the path %q: %v\n", target, err)
			return fmt.Errorf("cannot walk path %q: %w", target, err)
		}
	}

	if perr != nil && !hasTgzCharts {
		i.logger.Printf("error: cannot load chart: %s", perr)
		return perr
	}
	return nil
}

func (i *ImagesService) processTarget(target string) error {
	if i.verbose {
		i.logger.Printf("processig target: %s", target)
	}

	loadedChart, err := chartutil.Load(target)
	if err != nil {
		return fmt.Errorf("cannot load chart %q: %w", target, err)
	}
	caps := &chartutil.Capabilities{
		APIVersions:   chartutil.DefaultVersionSet,
		KubeVersion:   chartutil.DefaultKubeVersion,
		TillerVersion: tversion.GetVersionProto(),
	}
	chartConfig := &chart.Config{}
	vals, err := chartutil.ToRenderValuesCaps(loadedChart, chartConfig, renderutil.Options{}.ReleaseOptions, caps)
	if err != nil {
		i.logger.Printf("error: cannot render values: %s", err)
		return fmt.Errorf("cannot render chart %q values: %w", target, err)
	}

	vals = cleanUp(vals)
	renderer := engine.New()
	renderer.LintMode = i.ignoreErrors
	rendered, err := renderer.Render(loadedChart, vals)
	if err != nil {
		i.logger.Printf("error: cannot render chart: %s", err)
		return fmt.Errorf("cannot render chart %q: %w", target, err)
	}

	for _, t := range rendered {
		scanner := bufio.NewScanner(strings.NewReader(t))
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "image:") {
				im := sanitizeImageString(scanner.Text())
				i.buffer.WriteString(im + "\n")
			}
		}
	}
	return nil
}

func sanitizeImageString(str string) string {
	str = strings.Replace(str, "\"", "", 2)
	str = strings.TrimSpace(str)
	str = strings.TrimPrefix(str, "-")
	str = strings.TrimSpace(str)
	str = strings.TrimPrefix(str, "image: ")
	str = strings.TrimSpace(str)
	return str
}

func cleanUp(fields map[string]interface{}) map[string]interface{} {
	for key, val := range fields {
		switch a := val.(type) {
		case map[string]interface{}:
			fields[key] = cleanUp(a)
		case chartutil.Values:
			fields[key] = cleanUp(a)
		default:
			if val == nil {
				fields[key] = ""
			} else {
				fields[key] = val
			}
		}
	}
	return fields
}

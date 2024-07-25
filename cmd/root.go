// Copyright © 2018 openSUSE opensuse-project@opensuse.org
// Copyright © 2024 Patrick D'appollonio github@patrickdap.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/konstructio/helm-mirror/service"
	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/repo"
)

//nolint:gochecknoglobals
var (
	// Verbose defines if the command is being run with verbose mode
	Verbose bool
	// IgnoreErrors ignores errors in processing charts
	IgnoreErrors bool
	// AllVersions gets all the versions of the charts when true, false by default
	AllVersions  bool
	chartName    string
	chartVersion string
	folder       string
	flags        = log.Ldate | log.Lmicroseconds
	prefix       = "helm-mirror: "
	username     string
	password     string
	caFile       string
	certFile     string
	keyFile      string
	newRootURL   string
)

const rootDesc = `Mirror Helm Charts from an index file into a local folder.

For example:

	helm-mirror https://charts.example.com/ /path/to/downloaded/charts

This will download the index file and the latest version of the charts into the
folder indicated.

The index file is a YAML that contains a list of charts in this format. Example:

	apiVersion: v1
	entries:
	  chart:
	  - apiVersion: 1.0.0
	    created: 2018-08-08T00:00:00.00000000Z
	    description: A Helm chart for your application
	    digest: 3aa68d6cb66c14c1fcffc6dc6d0ad8a65b90b90c10f9f04125dc6fcaf8ef1b20
	    name: chart
	    urls:
	    - https://kubernetes-charts.example.com/chart-1.0.0.tgz
	  chart2:
	  - apiVersion: 1.0.0
	    created: 2018-08-08T00:00:00.00000000Z
	    description: A Helm chart for your application
	    digest: 7ae62d60b61c14c1fcffc6dc670e72e62b91b91c10f9f04125dc67cef2ef0b21
	    name: chart
	    urls:
	    - https://kubernetes-charts.example.com/chart2-1.0.0.tgz

This will download these charts

	https://kubernetes-charts.example.com/chart-1.0.0.tgz
	https://kubernetes-charts.example.com/chart2-1.0.0.tgz

Into your destination folder.`

// rootCmd represents the base command when called without any subcommands
//
//nolint:gochecknoglobals
var rootCmd = &cobra.Command{
	Use:   "mirror [Repo URL] [Destination Folder]",
	Short: "Mirror Helm Charts from an index file into a local folder.",
	Long:  rootDesc,
	Args:  validateRootArgs,
	RunE:  runRoot,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&IgnoreErrors, "ignore-errors", "i", false, "ignores errors while downloading or processing charts")
	rootCmd.PersistentFlags().BoolVarP(&AllVersions, "all-versions", "a", false, "gets all the versions of the charts in the chart repository")
	rootCmd.Flags().StringVar(&chartName, "chart-name", "", "name of the chart that gets mirrored")
	rootCmd.Flags().StringVar(&chartVersion, "chart-version", "", "specific version of the chart that is going to be mirrored")
	rootCmd.Flags().StringVar(&username, "username", "", "chart repository username")
	rootCmd.Flags().StringVar(&password, "password", "", "chart repository password")
	rootCmd.Flags().StringVar(&caFile, "ca-file", "", "verify certificates of HTTPS-enabled servers using this CA bundle")
	rootCmd.Flags().StringVar(&certFile, "cert-file", "", "identify HTTPS client using this SSL certificate file")
	rootCmd.Flags().StringVar(&keyFile, "key-file", "", "identify HTTPS client using this SSL key file")
	rootCmd.Flags().StringVar(&newRootURL, "new-root-url", "", "New root url of the chart repository (eg: `https://mirror.local.lan/charts`)")
	rootCmd.AddCommand(newVersionCmd())
}

func validateRootArgs(_ *cobra.Command, args []string) error {
	if len(args) < 2 {
		if len(args) == 1 && args[0] == "help" {
			return nil
		}
		return errors.New("error: requires at least two args to execute")
	}

	url, err := url.Parse(args[0])
	if err != nil {
		return fmt.Errorf("error: %q is not a valid URL for index file: %w", args[0], err)
	}

	if !strings.Contains(url.Scheme, "http") {
		return errors.New("error: not a valid URL protocol")
	}

	if !path.IsAbs(args[1]) {
		return errors.New("error: please provide a full path for destination folder")
	}

	return nil
}

func runRoot(_ *cobra.Command, args []string) error {
	logger := log.New(os.Stderr, prefix, flags)

	repoURL, err := url.Parse(args[0])
	if err != nil {
		logger.Printf("error: not a valid URL for index file: %s", err)
		return fmt.Errorf("error: %q is not a valid URL for index file: %w", args[0], err)
	}

	folder = args[1]
	if err := os.MkdirAll(folder, 0o744); err != nil {
		logger.Printf("error: cannot create destination folder: %s", err)
		return fmt.Errorf("cannot create destination folder %q: %w", folder, err)
	}

	rootURL := &url.URL{}
	if newRootURL != "" {
		rootURL, err = url.Parse(newRootURL)
		if err != nil {
			logger.Printf("error: new-root-url not a valid URL: %s", err)
			return fmt.Errorf("error: %q is not a valid URL: %w", newRootURL, err)
		}

		if !strings.Contains(rootURL.Scheme, "http") {
			logger.Printf("error: new-root-url not a valid URL protocol: `%s`", rootURL.Scheme)
			return errors.New("error: new-root-url not a valid URL protocol")
		}
	}

	if chartVersion != "" && chartName == "" {
		logger.Printf("error: chart Version depends on a chart name, please specify one")
		return errors.New("error: chart Version depends on a chart name, please specify one")
	}

	config := repo.Entry{
		Name:     folder,
		URL:      repoURL.String(),
		Username: username,
		Password: password,
		CAFile:   caFile,
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	getService := service.NewGetService(config, AllVersions, Verbose, IgnoreErrors, logger, rootURL.String(), chartName, chartVersion)
	if err := getService.Get(); err != nil {
		return fmt.Errorf("cannot download index and charts to the specified directory: %w", err)
	}

	return nil
}

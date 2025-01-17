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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/konstructio/helm-mirror/formatter"
	"github.com/konstructio/helm-mirror/service"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var (
	output string
	target string
)

const imagesDesc = `Extract all the images of the Helm Chart or
the Helm Charts in the folder provided. This command dumps
the images on 'stdout' by default, for more options check
'output' flag. Example:

  - helm mirror inspect-images /tmp/helm
  - helm mirror inspect-images /tmp/helm/app.tgz

The [folder|tgzfile] has to be a full path.
`

const outputDesc = `choose an output for the list of images and specify
the file name, if not specified 'images.out' will be the default.
Options:

- file: outputs all images to a file
- json: outputs all images to a file in JSON format
- skopeo: outputs all images to a file in YAML format
  to be used as source file with the 'skopeo sync' command.
  For more information refer to the 'skopeo sync'
  documentation at https://github.com/SUSE/skopeo/blob/sync/docs/skopeo.1.md#skopeo-sync
- stdout: prints all images to standard output
- yaml: outputs all images to a file in YAML format

Usage:

	- helm mirror inspect-images /tmp/helm --output stdout
	- helm mirror inspect-images /tmp/helm -o stdout
	- helm mirror inspect-images /tmp/helm -o file=filename
	- helm mirror inspect-images /tmp/helm -o json=filename.json
	- helm mirror inspect-images /tmp/helm -o yaml=filename.yaml
	- helm mirror inspect-images /tmp/helm -o skopeo=filename.yaml

`

// inspectImagesCmd represents the images command
//
//nolint:gochecknoglobals
var inspectImagesCmd = &cobra.Command{
	Use:   "inspect-images [folder|tgzfile]",
	Short: "Extract all the container images listed in each chart.",
	Long:  imagesDesc,
	Args:  validateInspectImagesArgs,
	RunE:  runInspectImages,
}

func init() {
	inspectImagesCmd.PersistentFlags().StringVarP(&output, "output", "o", "stdout", outputDesc)
	rootCmd.AddCommand(inspectImagesCmd)
}

func validateInspectImagesArgs(_ *cobra.Command, args []string) error {
	logger := log.New(os.Stderr, prefix, flags)

	if len(args) < 1 {
		logger.Print("error: requires at least one arg to execute")
		return errors.New("error: requires at least one arg")
	}

	if !path.IsAbs(args[0]) {
		logger.Printf("error: please provide a full path for [folder|tgzfile]: `%s`", args[0])
		return errors.New("error: please provide a full path for [folder|tgzfile]")
	}

	return nil
}

//nolint:ireturn
func resolveFormatter(output string, logger *log.Logger) (formatter.Formatter, error) {
	imagesFile := "images.out"

	pieces := strings.Split(output, "=")
	if len(pieces) > 1 {
		imagesFile = pieces[1]
	}

	imagesFile, err := filepath.Abs(imagesFile)
	if err != nil {
		logger.Print("error: getting working directory")
		return nil, fmt.Errorf("cannot get working directory: %w", err)
	}

	var ftype formatter.Type
	switch pieces[0] {
	case "file":
		ftype = formatter.FileType
	case "yaml":
		ftype = formatter.YamlType
	case "json":
		ftype = formatter.JSONType
	case "skopeo":
		ftype = formatter.SkopeoType
	default:
		ftype = formatter.StdoutType
	}

	return formatter.NewFormatter(ftype, imagesFile, logger), nil
}

func runInspectImages(_ *cobra.Command, args []string) error {
	logger := log.New(os.Stderr, prefix, flags)

	target = args[0]
	formatter, err := resolveFormatter(output, logger)
	if err != nil {
		return err
	}

	imagesService := service.NewImagesService(target, Verbose, IgnoreErrors, formatter, logger)
	if err := imagesService.Images(); err != nil {
		return fmt.Errorf("cannot extract images: %w", err)
	}

	return nil
}

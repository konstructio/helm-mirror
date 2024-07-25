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
	"fmt"

	"github.com/spf13/cobra"
)

// version is version ID for the source, read from VERSION in the source and
// populated on build by make.
var version = "unkwown"

// gitCommit is the commit hash that the binary was built from and will be
// populated on build by make.
//
//nolint:gochecknoglobals
var gitCommit = ""

// versionCmd represents the version command
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version of the helm mirror plugin",
		Run:   runVersion,
	}
}

func runVersion(*cobra.Command, []string) {
	ver := ""
	if version != "" {
		ver = version
	}
	if gitCommit != "" {
		ver = fmt.Sprintf("%s~git%s", ver, gitCommit)
	}
	fmt.Println(ver)
}

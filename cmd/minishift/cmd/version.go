/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/minishift/minishift/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Gets the version of Minishift.",
	Long:  `Gets the currently installed version of Minishift and prints it to standard output.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
	Run: runPrintVersion,
}

func runPrintVersion(cmd *cobra.Command, args []string) {
	// IMPORTANT - JBIDE uses the version string to integrate with Minishift/CDK. The format cannot change without discussion
	// See also https://issues.jboss.org/browse/CDK-150 and https://issues.jboss.org/browse/JBIDE-24682
	fmt.Println(fmt.Sprintf("minishift v%s+%s\nCDK v%s", version.GetMinishiftVersion(), version.GetCommitSha(), version.GetCDKVersion()))
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

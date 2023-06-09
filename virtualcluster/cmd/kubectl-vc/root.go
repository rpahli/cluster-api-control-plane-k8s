/*
Copyright 2020 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/version"
)

func main() {
	f, err := NewFactory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to new client factory: %v", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:     "kubectl-vc",
		Short:   "VirtualCluster Command tool",
		Version: version.BriefVersion(),
		RunE:    runHelp,
	}

	rootCmd.AddCommand(NewCmdCreate(f))
	rootCmd.AddCommand(NewCmdExec(f))

	CheckErr(rootCmd.Execute())
}

func runHelp(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

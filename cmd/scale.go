// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

var scaleTestOpts = &types.ScaleTestOpts{}

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:     "scale-test",
	Aliases: []string{"scale"},
	Short:   "Runs a load tests on a cluster",
	Long:    `This is a tool for running a scale test on a cluster by perfoming massive load on network and on pods.`,
	Run:     scaleRun,
}

func init() {
	PerfCmd.AddCommand(scaleCmd)
	scaleCmd.Flags().StringVarP(&scaleTestOpts.OutputDir, "output", "o", "./scaletest-results.csv", "Full path to result file to output")
	scaleCmd.Flags().BoolVarP(&scaleTestOpts.Cleanup, "cleanup", "c", true, "Delete test pods when done")
}

func scaleRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
	scaleTestOpts.Debug = RootOpts.Debug
	pkg.ScaleTest(config, scaleTestOpts)
}

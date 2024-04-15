/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taylormonacelli/eachit/run"
)

var (
	containerNamesToRemove []string
	excludeHcls            []string
	hclFiles               []string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Remove specified containers and build HCL files",
	Long:  `Remove the specified containers using the incus command and build HCL files using Packer.`,
	Run: func(cmd *cobra.Command, args []string) {
		run.BuildHclFiles(containerNamesToRemove, excludeHcls, hclFiles)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringSliceVar(&containerNamesToRemove, "destroy-containers", []string{"packer-jammy"}, "List of container names to remove")
	if err := runCmd.MarkFlagRequired("destroy-containers"); err != nil {
		fmt.Println("Error marking flag as required:", err)
		os.Exit(1)
	}

	runCmd.Flags().StringSliceVar(&excludeHcls, "exclude-hcls", []string{}, "List of HCL files to exclude from building")
	runCmd.Flags().StringSliceVar(&hclFiles, "hcl", []string{}, "List of HCL files to build (overrides default build list)")
}

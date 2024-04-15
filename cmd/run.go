package cmd

import (
	"github.com/spf13/cobra"
	"github.com/taylormonacelli/eachit/ntfy"
	"github.com/taylormonacelli/eachit/run"
)

var (
	containerNamesToRemove []string
	excludeHcls            []string
	hclFiles               []string
	ntfyChannel            string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Remove specified containers and build HCL files",
	Long:  `Remove the specified containers using the incus command and build HCL files using Packer.`,
	Run: func(cmd *cobra.Command, args []string) {
		ntfy.NtfyChannel = "https://ntfy.sh/" + ntfyChannel
		run.BuildHclFiles(containerNamesToRemove, excludeHcls, hclFiles)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringSliceVar(&containerNamesToRemove, "destroy-containers", []string{"mythai"}, "List of container names to remove")
	runCmd.Flags().StringSliceVar(&excludeHcls, "exclude-hcls", []string{}, "List of HCL files to exclude from building")
	runCmd.Flags().StringSliceVar(&hclFiles, "hcl", []string{}, "List of HCL files to build (overrides default build list)")
	runCmd.Flags().StringVar(&ntfyChannel, "ntfy-channel", "", "Ntfy channel for sending build notifications")
}

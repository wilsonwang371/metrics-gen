/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// gitPatchCmd represents the gitPatch command
var gitPatchCmd = &cobra.Command{
	Use:    "git-patch",
	Short:  "Patch code as git patch",
	Long:   `This command will generate a git patch that contains the patched code`,
	PreRun: PreRunGitPatch,
	Run:    RunGitPatch,
}

func init() {
	rootCmd.AddCommand(gitPatchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// gitPatchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// gitPatchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func PreRunGitPatch(cmd *cobra.Command, args []string) {
	// run root pre-run
	rootCmd.PreRun(cmd, args)
}

func RunGitPatch(cmd *cobra.Command, args []string) {
	panic("not implemented")
}

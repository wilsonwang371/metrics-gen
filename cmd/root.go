/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/parse"

	log "github.com/sirupsen/logrus"
)

var (
	// generatedFileSuffix string
	verbose             bool
	searchDirs          []string
	recursiveSearchDirs []string
	info                *parse.CollectInfo
	dryRun              bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "metrics-gen",
	Short: "Generate metrics capturing code for your Go project",
	Long: `This tool will parse your directive comments and generate new files
	that contain the code to capture the metrics for your code.`,
	PreRun: PreRunRoot,
	Run:    RunRoot,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
	// "config file (default is $HOME/.metrics-gen.yaml)")

	rootCmd.PersistentFlags().StringSliceVarP(&searchDirs, "dir", "d", []string{},
		"directory to search for files") // directory search option
	rootCmd.PersistentFlags().StringSliceVarP(&recursiveSearchDirs, "rdir",
		"r", []string{},
		"recursive directory to search for files") // recursive directory search option
	// rootCmd.PersistentFlags().StringVarP(&generatedFileSuffix, "suffix", "s",
	// 	"tracegen", ("suffix to add to generated files. If suffix is tracegen, then " +
	// 		"generated files will be named <filename>_tracegen.go")) // suffix option
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v",
		false, "verbose output") // verbose flag
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n",
		false, "dry run") // dry run flag
}

func RunRoot(cmd *cobra.Command, args []string) {
	// just print help and exit
	cmd.Help()
}

func PreRunRoot(cmd *cobra.Command, args []string) {
	// either dir or rdir must be specified
	if len(searchDirs) == 0 && len(recursiveSearchDirs) == 0 {
		log.Fatal("either dir or rdir must be specified")
	}

	// set log level
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// make sure we have go installed
	goVer := exec.Command("go", "version")
	err := goVer.Run()
	if err != nil {
		log.Fatal("go not found")
	}
}

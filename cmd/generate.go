/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/parse"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/platform"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/platform/common"
)

var (
	suffix        string
	inplace       bool
	provider      string
	metricsPrefix string // metrics names prefix, default to "metrics_gen"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generated new files with patched code",
	Long: `This command will generate new files with the patched code
that captures the metrics for your code.
	`,
	PreRun: PreRunGenerate,
	Run:    RunGenerate,
}

func needIgnore(filename string) bool {
	if inplace {
		// if inplace, we need not ignore any file
		return false
	}
	regex := regexp.MustCompile(fmt.Sprintf(`.*_%s\.go`, suffix))
	match := regex.MatchString(filename)
	if match {
		log.Debugf("file %s has suffix %s", filename, suffix)
	}
	return match
}

func addAllDirs() {
	if info == nil {
		log.Fatal("info is nil")
	}
	for _, dir := range searchDirs {
		err := info.AddTraceDir(dir, false, needIgnore)
		if err != nil {
			log.Fatalf("error adding dir %s: %v", dir, err)
		}
	}
	for _, dir := range recursiveSearchDirs {
		err := info.AddTraceDir(dir, true, needIgnore)
		if err != nil {
			log.Fatalf("error adding dir %s: %v", dir, err)
		}
	}

	// fail if no definition directive found anywhere in the files
	if !info.HasDefinitionDirective() {
		log.Fatal("no definition directive found")
	}
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// generateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// generateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	generateCmd.Flags().StringVarP(&suffix, "suffix", "s",
		"", ("suffix to add to generated files. If suffix is tracegen, then " +
			"generated files will be named <filename>_tracegen.go")) // suffix option
	generateCmd.Flags().BoolVarP(&inplace, "inplace", "i",
		false, "patch files in place") // inplace flag
	// provider choices
	generateCmd.Flags().StringVarP(&provider, "provider", "p", "prometheus",
		"metrics provider to use, supports \"gometrics\" & \"prometheus\"")
	generateCmd.Flags().StringVarP(&metricsPrefix, "metrics-prefix", "m",
		"metrics_gen", "generated metrics names prefix, default to \"metrics_gen\"")
}

func PreRunGenerate(cmd *cobra.Command, args []string) {
	// run root pre-run
	rootCmd.PreRun(cmd, args)

	// fail if suffix is not specified
	if suffix == "" && !inplace {
		log.Fatal("suffix must be specified")
	}

	// sufix and inplace are mutually exclusive
	if suffix != "" && inplace {
		log.Fatal("suffix and inplace are mutually exclusive")
	}
}

func RunGenerate(cmd *cobra.Command, args []string) {
	log.Debugf("dirs: %v. rdirs: %v", searchDirs,
		recursiveSearchDirs)

	info = parse.NewCollectInfo()
	addAllDirs()

	// select provider
	var p platform.MetricsProvider
	cfg := platform.MetricsProviderConfig{
		Inplace:       inplace,
		Suffix:        suffix,
		MetricsPrefix: metricsPrefix,
		Provider:      provider,
		DryRun:        dryRun,
	}

	if p = common.MetricsProviderFactory(cfg); p == nil {
		log.Fatalf("invalid provider %s", provider)
	}

	if err := p.PrePatch(info); err != nil {
		log.Fatalf("error pre patch: %v", err)
	}

	if err := p.Patch(info); err != nil {
		log.Fatalf("error patching: %v", err)
	}

	if err := p.PostPatch(info); err != nil {
		log.Fatalf("error post patch: %v", err)
	}
}

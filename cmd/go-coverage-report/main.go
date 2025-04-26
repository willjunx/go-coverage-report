package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/willjunx/go-coverage-report/pkg/config"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
	pkgReport "github.com/willjunx/go-coverage-report/pkg/report"
)

var usage = strings.TrimSpace(fmt.Sprintf(`
	Usage: %s [OPTIONS] <OLD_COVERAGE_FILE> <NEW_COVERAGE_FILE>
	
	Parse the OLD_COVERAGE_FILE and NEW_COVERAGE_FILE and compare the coverage of the
	changed files. The result is printed to stdout as a simple Markdown table with emojis 
	indicating the coverage change per package.
	
	You can use the -root flag to add a prefix to all paths in the list of changed
	files. This is useful to map the changed files (e.g., ["foo/my_file.go"] to their
	coverage profile which uses the full package name to identify the files
	(e.g., "github.com/username/example/foo/my_file.go"). Note that currently,
	packages with a different name than their directory are not supported.
	
	ARGUMENTS:
	  OLD_COVERAGE_FILE   The path to the old coverage file in the format produced by go test -coverprofile
	  NEW_COVERAGE_FILE   The path to the new coverage file in the same format as OLD_COVERAGE_FILE
	
	OPTIONS:
`, filepath.Base(os.Args[0])))

type options struct {
	root       string
	trim       string
	format     string
	configPath string
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		_, err := fmt.Fprintln(os.Stderr, usage)
		if err != nil {
			panic(err)
		}

		flag.PrintDefaults()
	}

	flag.String("root", "", "The import path of the tested repository to add as prefix to all paths of the changed files")
	flag.String("trim", "", "trim a prefix in the \"Impacted Packages\" column of the markdown report")
	flag.String("format", "markdown", "output format (currently only 'markdown' is supported)")
	flag.String("config", "", "path to the configuration file (.testcoverage.yaml), which defines test coverage settings and thresholds.")

	err := run(programArgs())
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
}

func programArgs() (oldCov, newCov string, opts options) {
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		if len(args) > 0 {
			log.Printf("ERROR: Expected exactly 2 arguments but got %d\n\n", len(args))
		}

		flag.Usage()
		os.Exit(1)
	}

	opts = options{
		root:       flag.Lookup("root").Value.String(),
		trim:       flag.Lookup("trim").Value.String(),
		format:     flag.Lookup("format").Value.String(),
		configPath: flag.Lookup("config").Value.String(),
	}

	return args[0], args[1], opts
}

func run(oldCovPath, newCovPath string, opts options) error {
	conf := config.Default
	if opts.configPath != "" {
		if err := config.FromFile(&conf, opts.configPath); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
	}

	oldCov, err := coverage.NewCoverageFromFile(oldCovPath)
	if err != nil {
		return fmt.Errorf("failed to parse old coverage: %w", err)
	}

	newCov, err := coverage.NewCoverageFromFile(newCovPath)
	if err != nil {
		return fmt.Errorf("failed to parse new coverage: %w", err)
	}

	changedFiles := pkgReport.GetChangedFiles(oldCov, newCov, conf.Exclude.Paths)
	if len(changedFiles) == 0 {
		log.Println("Skipping report since there are no changed files")
		return nil
	}

	conf.RootPackage = opts.root

	report := pkgReport.New(&conf, oldCov, newCov, changedFiles)
	if opts.trim != "" {
		report.TrimPrefix(opts.trim)
	}

	switch strings.ToLower(opts.format) {
	case "markdown":
		_, err = fmt.Fprintln(os.Stdout, report.Markdown())
		if err != nil {
			panic(err)
		}
	case "json":
		_, err = fmt.Fprintln(os.Stdout, report.JSON())
		if err != nil {
			panic(err)
		}
	default:
		return fmt.Errorf("unsupported format: %q", opts.format)
	}

	return nil
}

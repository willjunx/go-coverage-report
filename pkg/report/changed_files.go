package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
)

func GetChangedFiles(oldCov, newCov *coverage.Coverage, excludePaths []string) []string {
	var (
		oldFiles     = oldCov.Files
		newFiles     = newCov.Files
		res          = make([]string, 0, max(len(oldFiles), len(newFiles)))
		excludeRules = compileExcludePathRules(excludePaths)
	)

	for newFile, newProfile := range newFiles {
		if ok := matches(excludeRules, newFile); ok {
			continue // this file is excluded
		}

		oldProfile, ok := oldFiles[newFile]
		isNew := !ok
		isChange := newProfile.CoveragePercent() != oldProfile.CoveragePercent() ||
			len(newProfile.Blocks) != len(oldProfile.Blocks)

		if !isChange {
			for i, block := range newProfile.Blocks {
				if !block.Equal(oldProfile.Blocks[i]) {
					isChange = true
					break
				}
			}
		}

		if isNew || isChange {
			res = append(res, newFile)
		}
	}

	return res
}

func ParseChangedFiles(filename, prefix string) ([]string, error) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	var files []string
	err = json.Unmarshal(data, &files)

	if err != nil {
		return nil, err
	}

	for i, file := range files {
		files[i] = filepath.Join(prefix, file)
	}

	return files, nil
}

func compileExcludePathRules(excludePaths []string) []*regexp.Regexp {
	if len(excludePaths) == 0 {
		return nil
	}

	compiled := make([]*regexp.Regexp, len(excludePaths))

	for i, pattern := range excludePaths {
		compiled[i] = regexp.MustCompile(pattern)
	}

	return compiled
}

func matches(regexps []*regexp.Regexp, str string) bool {
	for _, r := range regexps {
		if r.MatchString(str) {
			return true
		}
	}

	return false
}

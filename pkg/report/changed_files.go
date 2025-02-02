package report

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
)

func GetChangedFiles(oldCov, newCov *coverage.Coverage) []string {
	var (
		oldFiles = oldCov.Files
		newFiles = newCov.Files
		res      = make([]string, 0, max(len(oldFiles), len(newFiles)))
	)

	for newFile, newProfile := range newFiles {
		oldProfile, ok := oldFiles[newFile]
		isNew := !ok
		isChange := newProfile.CoveragePercent() != oldProfile.CoveragePercent() ||
			len(newProfile.Blocks) != len(oldProfile.Blocks)

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

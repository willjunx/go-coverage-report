package report

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
)

type Report struct {
	Old, New        *coverage.Coverage
	ChangedFiles    []string
	ChangedPackages []string
}

func New(oldCov, newCov *coverage.Coverage, changedFiles []string) *Report {
	sort.Strings(changedFiles)

	return &Report{
		Old:             oldCov,
		New:             newCov,
		ChangedFiles:    changedFiles,
		ChangedPackages: changedPackages(changedFiles),
	}
}

func changedPackages(changedFiles []string) []string {
	var (
		res     = make([]string, 0)
		visited = make(map[string]bool)
	)

	for _, file := range changedFiles {
		pkg := filepath.Dir(file)
		if visited[pkg] {
			continue
		}

		res = append(res, pkg)
		visited[pkg] = true
	}

	sort.Strings(res)

	return res
}

func (r *Report) Markdown() string {
	report := new(strings.Builder)

	_, _ = fmt.Fprintln(report, r.Title())
	_, _ = fmt.Fprintln(report, "| Impacted Packages | Coverage Δ | :robot: |")
	_, _ = fmt.Fprintln(report, "|-------------------|------------|---------|")

	oldCovPkgs := r.Old.ByPackage()
	newCovPkgs := r.New.ByPackage()

	for _, pkg := range r.ChangedPackages {
		var oldPercent, newPercent float64

		if cov, ok := oldCovPkgs[pkg]; ok {
			oldPercent = cov.Percent()
		}

		if cov, ok := newCovPkgs[pkg]; ok {
			newPercent = cov.Percent()
		}

		emoji, diffStr := emojiScore(newPercent, oldPercent)
		_, _ = fmt.Fprintf(report, "| %s | %.2f%% (%s) | %s |\n",
			pkg, newPercent, diffStr, emoji,
		)
	}

	report.WriteString("\n")
	r.addDetails(report)

	return report.String()
}

func (r *Report) JSON() string {
	data, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		panic(err) // should never happen
	}

	return string(data)
}

func (r *Report) Title() string {
	var (
		precision                = 2
		oldCovPkgs, newCovPkgs   = r.Old.ByPackage(), r.New.ByPackage()
		numDecrease, numIncrease int
	)

	for _, pkg := range r.ChangedPackages {
		oldPercent, newPercent := 0.0, 0.0

		if cov, ok := oldCovPkgs[pkg]; ok {
			oldPercent = roundFloat(cov.Percent(), precision)
		}

		if cov, ok := newCovPkgs[pkg]; ok {
			newPercent = roundFloat(cov.Percent(), precision)
		}

		if newPercent > oldPercent {
			numIncrease++
		} else if newPercent < oldPercent {
			numDecrease++
		}
	}

	title := fmt.Sprintf("## Coverage Percentage %.2f%%\n", r.New.Percent())

	switch {
	case numIncrease == 0 && numDecrease == 0:
		title += fmt.Sprintln("### Merging this branch will **not change** overall coverage")
	case numIncrease > 0 && numDecrease == 0:
		title += fmt.Sprintln("### Merging this branch will **increase** overall coverage")
	case numIncrease == 0 && numDecrease > 0:
		title += fmt.Sprintln("### Merging this branch will **decrease** overall coverage")
	default:
		title += fmt.Sprintf("### Merging this branch changes the coverage (%d decrease, %d increase)\n", numDecrease, numIncrease)
	}

	return title
}

func (r *Report) addDetails(report *strings.Builder) {
	_, _ = fmt.Fprintln(report, "---")
	_, _ = fmt.Fprintln(report)
	_, _ = fmt.Fprintln(report, "<details>")
	_, _ = fmt.Fprintln(report)

	_, _ = fmt.Fprintln(report, "<summary>Coverage by file</summary>")
	_, _ = fmt.Fprintln(report)

	var codeFiles, unitTestFiles []string

	for _, f := range r.ChangedFiles {
		if strings.HasSuffix(f, "_test.go") {
			unitTestFiles = append(unitTestFiles, f)
		} else {
			codeFiles = append(codeFiles, f)
		}
	}

	if len(codeFiles) > 0 {
		r.addCodeFileDetails(report, codeFiles)
	}

	if len(unitTestFiles) > 0 {
		r.addChangedTestFileDetails(report, unitTestFiles)
	}

	_, _ = fmt.Fprint(report, "</details>")
}

func (r *Report) addCodeFileDetails(report *strings.Builder, files []string) {
	_, _ = fmt.Fprintln(report, "### Changed files (no unit tests)")
	_, _ = fmt.Fprintln(report)
	_, _ = fmt.Fprintln(report, "| Changed File | Coverage Δ | Total | Covered | Missed | :robot: |")
	_, _ = fmt.Fprintln(report, "|--------------|------------|-------|---------|--------|---------|")

	for _, name := range files {
		oldProfile, newProfile := r.Old.Files[name], r.New.Files[name]
		oldPercent, newPercent := oldProfile.CoveragePercent(), newProfile.CoveragePercent()

		valueWithDelta := func(oldVal, newVal int) string {
			diff := oldVal - newVal

			switch {
			case diff < 0:
				return fmt.Sprintf("%d (+%d)", newVal, -diff)
			case diff > 0:
				return fmt.Sprintf("%d (-%d)", newVal, diff)
			default:
				return fmt.Sprintf("%d", newVal)
			}
		}

		emoji, diffStr := emojiScore(newPercent, oldPercent)
		_, _ = fmt.Fprintf(report, "| %s | %.2f%% (%s) | %s | %s | %s | %s |\n",
			name,
			newPercent, diffStr,
			valueWithDelta(oldProfile.GetTotal(), newProfile.GetTotal()),
			valueWithDelta(oldProfile.GetCovered(), newProfile.GetCovered()),
			valueWithDelta(oldProfile.GetMissed(), newProfile.GetMissed()),
			emoji,
		)
	}
}

func (r *Report) addChangedTestFileDetails(report *strings.Builder, files []string) {
	_, _ = fmt.Fprintln(report, "### Changed unit test files")
	_, _ = fmt.Fprintln(report)

	for _, name := range files {
		_, _ = fmt.Fprintf(report, "- %s\n", name)
	}

	_, _ = fmt.Fprintln(report)
}

func roundFloat(val float64, precision int) float64 {
	if val == 0 {
		return 0
	}

	pow := math.Pow10(precision)

	return math.Round(pow*val) / pow
}

func emojiScore(newPercent, oldPercent float64) (emoji, diffStr string) {
	diff := newPercent - oldPercent
	diffStr = fmt.Sprintf("**%+.2f%%**", diff)

	switch {
	case diff < -50:
		emoji = strings.Repeat(":skull: ", 5)
	case diff < -10:
		emoji = strings.Repeat(":skull: ", int(-diff/10))
	case diff < 0:
		emoji = ":thumbsdown:"
	case diff == 0:
		emoji = ""
		diffStr = "ø"
	case diff > 20:
		emoji = ":star2:"
	case diff > 10:
		emoji = ":tada:"
	case diff > 0:
		emoji = ":thumbsup:"
	}

	return emoji, diffStr
}

func (r *Report) TrimPrefix(prefix string) {
	for i, name := range r.ChangedPackages {
		r.ChangedPackages[i] = coverage.TrimPrefix(name, prefix)
	}

	for i, name := range r.ChangedFiles {
		r.ChangedFiles[i] = coverage.TrimPrefix(name, prefix)
	}

	r.Old.TrimPrefix(prefix)
	r.New.TrimPrefix(prefix)
}

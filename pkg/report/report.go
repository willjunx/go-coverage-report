package report

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/willjunx/go-coverage-report/pkg/config"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
)

type Report struct {
	Old, New        *coverage.Coverage
	ChangedFiles    []string
	ChangedPackages []string

	PackageCoveragePass CoveragePass
	FileCoveragePass    CoveragePass
	TotalCoveragePass   bool

	conf *config.Config
}

func New(conf *config.Config, oldCov, newCov *coverage.Coverage, changedFiles []string) *Report {
	sort.Strings(changedFiles)
	curChangedPackages := changedPackages(changedFiles)

	return &Report{
		Old:                 oldCov,
		New:                 newCov,
		ChangedFiles:        changedFiles,
		ChangedPackages:     curChangedPackages,
		PackageCoveragePass: checkPackageCoverage(conf.Threshold.Package, newCov, curChangedPackages),
		FileCoveragePass:    checkFileCoverage(conf.Threshold.File, newCov, changedFiles),
		TotalCoveragePass:   isCoveragePassed(conf.Threshold.Total, newCov.Percent()),
		conf:                conf,
	}
}

func checkFileCoverage(threshold int, cov *coverage.Coverage, changedFiles []string) CoveragePass {
	var res = CoveragePass{
		Value:  true,
		Detail: make(map[string]bool),
	}

	if threshold <= 0 {
		return res
	}

	for _, filename := range changedFiles {
		fileCov, ok := cov.Files[filename]
		if !ok {
			continue
		}

		res.Detail[filename] = isCoveragePassed(threshold, fileCov.CoveragePercent())
		if !res.Detail[filename] {
			res.Value = false
		}
	}

	return res
}

func checkPackageCoverage(threshold int, cov *coverage.Coverage, changedPackages []string) CoveragePass {
	var res = CoveragePass{
		Value:  true,
		Detail: make(map[string]bool),
	}

	if threshold <= 0 {
		return res
	}

	packages := cov.ByPackage()

	for _, pkg := range changedPackages {
		pkgCov, ok := packages[pkg]
		if !ok {
			continue
		}

		res.Detail[pkg] = isCoveragePassed(threshold, pkgCov.Percent())
		if !res.Detail[pkg] {
			res.Value = false
		}
	}

	return res
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
	var (
		report           = new(strings.Builder)
		hasCheckCoverage = r.conf.Threshold.Package > 0
	)

	_, _ = fmt.Fprintln(report, r.Title())

	var (
		header    = "| Impacted Packages | Coverage Δ | :robot: |"
		separator = "|-------------------|------------|---------|"
	)

	if hasCheckCoverage {
		header += " Pass |"
		separator += "------|"
	}

	_, _ = fmt.Fprintln(report, header)
	_, _ = fmt.Fprintln(report, separator)

	oldCovPkgs := r.Old.ByPackage()
	newCovPkgs := r.New.ByPackage()
	fmt.Println("HALOHA", r.PackageCoveragePass.Detail)
	for _, pkg := range r.ChangedPackages {
		var oldPercent, newPercent float64

		if cov, ok := oldCovPkgs[pkg]; ok {
			oldPercent = cov.Percent()
		}

		if cov, ok := newCovPkgs[pkg]; ok {
			newPercent = cov.Percent()
		}

		emoji, diffStr := emojiScore(newPercent, oldPercent)

		format := "| %s | %.2f%% (%s) | %s |"
		args := []interface{}{filepath.Join(r.conf.RootPackage, pkg), newPercent, diffStr, emoji}

		if hasCheckCoverage {
			format += " %s |"
			fmt.Println("FAKKA", pkg)
			args = append(args, emojiPass(r.PackageCoveragePass.Detail[pkg]))
		}

		_, _ = fmt.Fprintf(report, format+"\n", args...)
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

	if r.conf.Threshold.Total > 0 || r.conf.Threshold.File > 0 || r.conf.Threshold.Package > 0 {
		_, _ = fmt.Fprintln(report)
		r.addTotalCoverageResult(report)
	}
}

func (r *Report) addTotalCoverageResult(report *strings.Builder) {
	_, _ = fmt.Fprintln(report, "\n---")

	pass := r.TotalCoveragePass && r.PackageCoveragePass.Value && r.FileCoveragePass.Value

	_, _ = fmt.Fprintf(
		report,
		"### Coverage Result: %s %s",
		emojiPass(pass),
		func() string {
			if pass {
				return "PASS"
			}

			return "FAIL"
		}(),
	)
}

func (r *Report) addCodeFileDetails(report *strings.Builder, files []string) {
	_, _ = fmt.Fprintln(report, "### Changed files")
	_, _ = fmt.Fprintln(report)

	var (
		header    = "| Changed File | Coverage Δ | Total | Covered | Missed | :robot: |"
		separator = "|--------------|------------|-------|---------|--------|---------|"
	)

	hasCheck := r.conf.Threshold.File > 0
	if hasCheck {
		header += " Pass |"
		separator += "------|"
	}

	_, _ = fmt.Fprintln(report, header)
	_, _ = fmt.Fprintln(report, separator)

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

		format := "| %s | %.2f%% (%s) | %s | %s | %s | %s |"
		args := []any{
			filepath.Join(r.conf.RootPackage, name),
			newPercent, diffStr,
			valueWithDelta(oldProfile.GetTotal(), newProfile.GetTotal()),
			valueWithDelta(oldProfile.GetCovered(), newProfile.GetCovered()),
			valueWithDelta(oldProfile.GetMissed(), newProfile.GetMissed()),
			emoji,
		}

		if hasCheck {
			format += " %s |"

			args = append(args, emojiPass(r.FileCoveragePass.Detail[name]))
		}

		_, _ = fmt.Fprintf(report, format+"\n", args...)
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

func emojiPass(val bool) string {
	if val {
		return ":white_check_mark:"
	}

	return ":negative_squared_cross_mark:"
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

func isCoveragePassed(threshold int, cov float64) bool {
	if threshold == 0 {
		return true
	}

	return int(math.Ceil(cov)) >= threshold
}

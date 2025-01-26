package src_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/willjunx/go-coverage-report/src"
)

var _ = Describe("Report", func() {
	Context("Markdown", func() {
		It("Should return correctly", func() {
			oldCov, err := src.NewCoverageFromFile("testdata/01-old-coverage.txt")
			Expect(err).ToNot(HaveOccurred())

			newCov, err := src.NewCoverageFromFile("testdata/01-new-coverage.txt")
			Expect(err).ToNot(HaveOccurred())

			changedFiles, err := src.ParseChangedFiles("testdata/01-changed-files.json", "github.com/fgrosse/prioqueue")
			Expect(err).ToNot(HaveOccurred())

			report := src.NewReport(oldCov, newCov, changedFiles)
			actual := report.Markdown()

			expected := `### Merging this branch will **decrease** overall coverage

| Impacted Packages | Coverage Δ | :robot: |
|-------------------|------------|---------|
| github.com/fgrosse/prioqueue | 90.20% (**-9.80%**) | :thumbsdown: |
| github.com/fgrosse/prioqueue/foo/bar | 0.00% (ø) |  |

---

<details>

<summary>Coverage by file</summary>

### Changed files (no unit tests)

| Changed File | Coverage Δ | Total | Covered | Missed | :robot: |
|--------------|------------|-------|---------|--------|---------|
| github.com/fgrosse/prioqueue/foo/bar/baz.go | 0.00% (ø) | 0 | 0 | 0 |  |
| github.com/fgrosse/prioqueue/min_heap.go | 80.77% (**-19.23%**) | 52 (+2) | 42 (-8) | 10 (+10) | :skull:  |
</details>`

			Expect(actual).To(Equal(expected))
		})

		When("Only Changed Unit Tests", func() {
			It("Should return correctly", func() {
				oldCov, err := src.NewCoverageFromFile("testdata/02-old-coverage.txt")
				Expect(err).ToNot(HaveOccurred())

				newCov, err := src.NewCoverageFromFile("testdata/02-new-coverage.txt")
				Expect(err).ToNot(HaveOccurred())

				changedFiles, err := src.ParseChangedFiles("testdata/02-changed-files.json", "github.com/fgrosse/prioqueue")
				Expect(err).ToNot(HaveOccurred())

				report := src.NewReport(oldCov, newCov, changedFiles)
				actual := report.Markdown()

				expected := `### Merging this branch will **increase** overall coverage

| Impacted Packages | Coverage Δ | :robot: |
|-------------------|------------|---------|
| github.com/fgrosse/prioqueue | 99.02% (**+8.82%**) | :thumbsup: |

---

<details>

<summary>Coverage by file</summary>

### Changed unit test files

- github.com/fgrosse/prioqueue/min_heap_test.go

</details>`

				Expect(actual).To(Equal(expected))
			})
		})
	})
})

package report_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
	"github.com/willjunx/go-coverage-report/pkg/report"
)

var _ = Describe("Report", func() {
	Context("GetChangedFiles", func() {
		It("Should return correctly", func() {
			oldCov, err := coverage.NewCoverageFromFile("testdata/01-old-coverage.txt")
			Expect(err).ToNot(HaveOccurred())

			newCov, err := coverage.NewCoverageFromFile("testdata/01-new-coverage.txt")
			Expect(err).ToNot(HaveOccurred())

			changedFiles := report.GetChangedFiles(oldCov, newCov, nil)
			Expect(changedFiles).To(Equal([]string{"github.com/username/prioqueue/min_heap.go"}))
		})

		When("with exclude paths", func() {
			It("should return correctly", func() {
				oldCov, err := coverage.NewCoverageFromFile("testdata/02-old-coverage.txt")
				Expect(err).ToNot(HaveOccurred())

				newCov, err := coverage.NewCoverageFromFile("testdata/02-new-coverage.txt")
				Expect(err).ToNot(HaveOccurred())

				changedFiles := report.GetChangedFiles(oldCov, newCov, []string{"^github.com/username"})
				Expect(changedFiles).To(Equal([]string{}))
			})
		})
	})

})

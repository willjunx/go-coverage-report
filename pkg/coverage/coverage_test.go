package coverage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/willjunx/go-coverage-report/pkg/coverage"
)

var _ = Describe("Coverage", func() {

	Context("NewCoverageFromFile", func() {
		It("Should return correctly", func() {
			cov, err := coverage.NewCoverageFromFile("testdata/01-new-coverage.txt")
			Expect(err).NotTo(HaveOccurred())

			Expect(cov.TotalStmt).To(Equal(102))
			Expect(cov.CoveredStmt).To(Equal(92))
			Expect(cov.MissedStmt).To(Equal(10))
			Expect(cov.Percent()).To(BeNumerically("~", 90.196, 0.001))
		})

		When("Filter by Package", func() {
			It("Should return correctly", func() {
				cov, err := coverage.NewCoverageFromFile("testdata/01-new-coverage.txt")
				Expect(err).NotTo(HaveOccurred())

				pkgs := cov.ByPackage()
				Expect(pkgs).To(HaveLen(1))

				pkgCov := pkgs["github.com/fgrosse/prioqueue"]
				Expect(pkgCov).ToNot(BeNil())
				Expect(pkgCov.TotalStmt).To(Equal(102))
				Expect(pkgCov.CoveredStmt).To(Equal(92))
				Expect(pkgCov.MissedStmt).To(Equal(10))
			})
		})
	})
})

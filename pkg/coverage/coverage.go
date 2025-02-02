package coverage

import (
	"fmt"
	"path"
)

type Coverage struct {
	Files       map[string]Profile
	TotalStmt   int
	CoveredStmt int
	MissedStmt  int
}

func NewCoverage(profiles []Profile) *Coverage {
	cov := Coverage{Files: map[string]Profile{}}
	for _, p := range profiles {
		cov.add(p)
	}

	return &cov
}

func NewCoverageFromFile(filename string) (*Coverage, error) {
	pp, err := NewProfilesFromFile(filename)
	if err != nil {
		return nil, err
	}

	return NewCoverage(pp), nil
}

func (c *Coverage) add(p Profile) {
	if _, ok := c.Files[p.FileName]; ok {
		panic(fmt.Errorf("profile for file %q already exists", p.FileName))
	}

	c.Files[p.FileName] = p
	c.TotalStmt += p.TotalStmt
	c.CoveredStmt += p.CoveredStmt
	c.MissedStmt += p.MissedStmt
}

func (c *Coverage) Percent() float64 {
	if c.TotalStmt == 0 {
		return 0
	}

	return float64(c.CoveredStmt) / float64(c.TotalStmt) * 100
}

func (c *Coverage) ByPackage() map[string]*Coverage {
	packages := map[string][]string{} // maps package paths to files

	for file := range c.Files {
		pkg := path.Dir(file)
		packages[pkg] = append(packages[pkg], file)
	}

	pkgCovs := make(map[string]*Coverage, len(packages))

	for pkg, files := range packages {
		var profiles []Profile
		for _, file := range files {
			profiles = append(profiles, c.Files[file])
		}

		pkgCovs[pkg] = NewCoverage(profiles)
	}

	return pkgCovs
}

func (c *Coverage) TrimPrefix(prefix string) {
	for name, cov := range c.Files {
		delete(c.Files, cov.FileName)
		cov.FileName = TrimPrefix(name, prefix)
		c.Files[cov.FileName] = cov
	}
}

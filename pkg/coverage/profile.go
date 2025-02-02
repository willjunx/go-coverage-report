package coverage

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Profile represents the profiling data for a specific file.
type Profile struct {
	FileName string `json:"-"`
	Mode     string
	Blocks   []ProfileBlock `json:"-"`

	TotalStmt   int
	CoveredStmt int
	MissedStmt  int
}

// ProfileBlock represents a single block of profiling data.
type ProfileBlock struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	NumStmt             int
	ExecCount           int
}

func (p ProfileBlock) Equal(b ProfileBlock) bool {
	return p.StartLine == b.StartLine &&
		p.StartCol == b.StartCol &&
		p.EndLine == b.EndLine &&
		p.EndCol == b.EndCol &&
		p.NumStmt == b.NumStmt
}

// NewProfilesFromFile parses profile data in the specified file and returns a
// Profile for each source file described therein.
func NewProfilesFromFile(fileName string) ([]Profile, error) {
	pf, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = pf.Close()
	}()

	return ParseProfilesFromReader(pf)
}

func (p Profile) CoveragePercent() float64 {
	if p.TotalStmt == 0 {
		return 0
	}

	return float64(p.CoveredStmt) / float64(p.TotalStmt) * 100
}

func (p Profile) GetTotal() int {
	return p.TotalStmt
}

func (p Profile) GetCovered() int {
	return p.CoveredStmt
}

func (p Profile) GetMissed() int {
	return p.MissedStmt
}

// ParseProfilesFromReader parses profile data from the Reader and
// returns a Profile for each source file described therein.
func ParseProfilesFromReader(rd io.Reader) ([]Profile, error) { //nolint:funlen,gocognit // expected
	var (
		mode  string
		files = make(map[string]Profile)
		s     = bufio.NewScanner(rd)
	)

	for s.Scan() {
		line := s.Text()

		if isFirstLine := mode == ""; isFirstLine {
			const p = "mode: "

			if !strings.HasPrefix(line, p) || line == p {
				return nil, fmt.Errorf("bad mode line: %v", line)
			}

			mode = line[len(p):]

			continue
		}

		fileName, block, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %q doesn't match expected format: %v", line, err)
		}

		if profile, exist := files[fileName]; !exist {
			files[fileName] = Profile{
				FileName: fileName,
				Mode:     mode,
				Blocks:   []ProfileBlock{block},
			}
		} else {
			profile.Blocks = append(profile.Blocks, block)
			files[fileName] = profile
		}
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(files))

	for _, p := range files {
		sort.Sort(blocksByStart(p.Blocks))

		n := len(p.Blocks)
		newBlocks := make([]ProfileBlock, 0)

		for l, r := 0, 0; l < n; l++ {
			r = l
			startLine, endLine := p.Blocks[l].StartLine, p.Blocks[l].EndLine
			startCol, endCol := p.Blocks[l].StartCol, p.Blocks[l].EndCol
			curBlock := p.Blocks[l]
			execCount := curBlock.ExecCount

			for r+1 < n && (startLine == p.Blocks[r+1].StartLine && endLine == p.Blocks[r+1].EndLine &&
				startCol == p.Blocks[r+1].StartCol && endCol == p.Blocks[r+1].EndCol) {
				nextBlock := p.Blocks[r+1]

				if nextBlock.NumStmt != curBlock.NumStmt {
					return nil, fmt.Errorf("inconsistent NumStmt: changed from %d to %d", curBlock.NumStmt, nextBlock.NumStmt)
				}

				if mode == "set" {
					execCount |= nextBlock.ExecCount
				} else {
					execCount += nextBlock.ExecCount
				}

				r++
			}

			curBlock.ExecCount = execCount
			newBlocks = append(newBlocks, curBlock)
			l = r
		}

		for _, b := range newBlocks {
			p.TotalStmt += b.NumStmt

			if b.ExecCount > 0 {
				p.CoveredStmt += b.NumStmt
			}
		}

		p.MissedStmt = p.TotalStmt - p.CoveredStmt

		p.Blocks = newBlocks

		profiles = append(profiles, p)
	}

	sort.Sort(byFileName(profiles))

	return profiles, nil
}

// parseLine parses a line from a coverage file.
func parseLine(l string) (fileName string, block ProfileBlock, err error) {
	var (
		b   ProfileBlock
		end = len(l)
	)

	b.ExecCount, end, err = seekBack(l, ' ', end, "ExecCount")
	if err != nil {
		return "", b, err
	}

	b.NumStmt, end, err = seekBack(l, ' ', end, "NumStmt")
	if err != nil {
		return "", b, err
	}

	b.EndCol, end, err = seekBack(l, '.', end, "EndCol")
	if err != nil {
		return "", b, err
	}

	b.EndLine, end, err = seekBack(l, ',', end, "EndLine")
	if err != nil {
		return "", b, err
	}

	b.StartCol, end, err = seekBack(l, '.', end, "StartCol")
	if err != nil {
		return "", b, err
	}

	b.StartLine, end, err = seekBack(l, ':', end, "StartLine")
	if err != nil {
		return "", b, err
	}

	fileName = l[:end]
	if fileName == "" {
		return "", b, errors.New("a FileName cannot be blank")
	}

	return fileName, b, nil
}

// seekBack searches backwards from end to find sep in l, then returns the
// value between sep and end as an integer.
// If seekBack fails, the returned error will reference `what`.
func seekBack(l string, sep byte, end int, what string) (value, nextSep int, err error) {
	for cur := end - 1; cur >= 0; cur-- {
		if l[cur] == sep {
			i, err := strconv.Atoi(l[cur+1 : end])
			if err != nil {
				return 0, 0, fmt.Errorf("couldn't parse %q: %v", what, err)
			}

			if i < 0 {
				return 0, 0, fmt.Errorf("negative values are not allowed for %s, found %d", what, i)
			}

			return i, cur, nil
		}
	}

	return 0, 0, fmt.Errorf("couldn't find a %s before %s", string(sep), what)
}

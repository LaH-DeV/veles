package sema

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a package version, `major.minor.patch` with an optional
// `-pre-release` (D132: versions are tags; a leading `v` is allowed in a
// tag and dropped here). Build metadata (`+…`) is refused: two versions
// that differ only there would be one tag.
type Version struct {
	Major, Minor, Patch uint64
	Pre                 string
}

// ParseVersion reads `1.4.2`, `v1.4.2` or `1.4.2-rc.1`.
func ParseVersion(s string) (Version, error) {
	orig := s
	s = strings.TrimPrefix(s, "v")
	var v Version
	if i := strings.IndexByte(s, '-'); i >= 0 {
		v.Pre = s[i+1:]
		s = s[:i]
		if v.Pre == "" || !validPre(v.Pre) {
			return v, detail("%q: the pre-release part after '-' is letters, digits, '.' and '-'", orig)
		}
	}
	if strings.Contains(s, "+") {
		return v, detail("%q: build metadata ('+…') is not used in versions", orig)
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, detail("%q is not a version: write major.minor.patch, e.g. 1.4.2", orig)
	}
	var nums [3]uint64
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 32)
		if err != nil || (len(part) > 1 && part[0] == '0') {
			return v, detail("%q is not a version: %q is not a number without leading zeros", orig, part)
		}
		nums[i] = n
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

func validPre(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-') {
			return false
		}
	}
	return true
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// MajorID is what makes two versions of one package the same package
// (D138, Cargo's rule): the first non-zero component, so `1.4.2` is "1",
// `0.3.1` is "0.3" and `0.0.2` is "0.0.2". Versions with equal MajorID
// are compatible; the minimal version selection picks the highest one.
func (v Version) MajorID() string {
	switch {
	case v.Major > 0:
		return strconv.FormatUint(v.Major, 10)
	case v.Minor > 0:
		return fmt.Sprintf("0.%d", v.Minor)
	}
	return fmt.Sprintf("0.0.%d", v.Patch)
}

// Compare orders versions: -1, 0 or 1. A pre-release sorts before its
// release; pre-releases compare by dot-separated parts, numbers as numbers
// and before words.
func (v Version) Compare(o Version) int {
	for _, p := range [][2]uint64{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Patch, o.Patch}} {
		if p[0] != p[1] {
			if p[0] < p[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case v.Pre == o.Pre:
		return 0
	case v.Pre == "":
		return 1
	case o.Pre == "":
		return -1
	}
	a, b := strings.Split(v.Pre, "."), strings.Split(o.Pre, ".")
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		x, xe := strconv.ParseUint(a[i], 10, 64)
		y, ye := strconv.ParseUint(b[i], 10, 64)
		switch {
		case xe == nil && ye == nil:
			if x < y {
				return -1
			}
			return 1
		case xe == nil:
			return -1
		case ye == nil:
			return 1
		case a[i] < b[i]:
			return -1
		}
		return 1
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// detail makes the reason half of a message: callers put the file and line
// in front of it ("veles.toml:12: dependency 'x': <detail>"), so it carries
// no location of its own.
func detail(format string, a ...any) error { return fmt.Errorf(format, a...) }

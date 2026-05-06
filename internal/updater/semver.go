package updater

import (
	"strconv"
	"strings"
)

type versionParts struct {
	major int
	minor int
	patch int
	ok    bool
}

func isSemver(v string) bool {
	return parseVersion(v).ok
}

func less(a, b string) bool {
	av := parseVersion(a)
	bv := parseVersion(b)
	if !av.ok || !bv.ok {
		return false
	}
	if av.major != bv.major {
		return av.major < bv.major
	}
	if av.minor != bv.minor {
		return av.minor < bv.minor
	}
	return av.patch < bv.patch
}

func parseVersion(v string) versionParts {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	main, _, _ := strings.Cut(v, "-")
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return versionParts{}
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return versionParts{}
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return versionParts{}
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return versionParts{}
	}
	return versionParts{major: major, minor: minor, patch: patch, ok: true}
}

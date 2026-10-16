package digreflect

import (
	"net/url"
	"strings"
)

const _vendor = "/vendor/"

func splitFuncName(function string) (pkgName string, funcName string) {
	if len(function) == 0 {
		return
	}

	// We have something like "path.to/my/pkg.MyFunction". If the function is
	// a closure, it is something like, "path.to/my/pkg.MyFunction.func1".

	idx := 0

	// Everything up to the first "." after the last "/" is the package name.
	// Everything after the "." is the full function name.
	if i := strings.LastIndex(function, "/"); i >= 0 {
		idx = i
	}
	if i := strings.Index(function[idx:], "."); i >= 0 {
		idx += i
	}
	pkgName, funcName = function[:idx], function[idx+1:]

	// The package may be vendored.
	if i := strings.Index(pkgName, _vendor); i > 0 {
		pkgName = pkgName[i+len(_vendor):]
	}

	// Package names are URL-encoded to avoid ambiguity in the case where the
	// package name contains ".git". Otherwise, "foo/bar.git.MyFunction" would
	// mean that "git" is the top-level function and "MyFunction" is embedded
	// inside it.
	if unescaped, err := url.QueryUnescape(pkgName); err == nil {
		pkgName = unescaped
	}

	return
}

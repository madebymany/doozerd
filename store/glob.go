package store

import (
	"regexp"
	"strings"
)

// Glob holds a Unix-style glob pattern in a compiled form for efficient
// matching against paths.
//
// Glob notation:
//  - `?` matches a single char in a single path component
//  - `*` matches zero or more chars in a single path component
//  - `**` matches zero or more chars in zero or more components
//  - any other sequence matches itself
type Glob struct {
	Pattern string         // original glob pattern
	s       string         // translated to regexp pattern
	r       *regexp.Regexp // compiled regexp
}

var globRePart = `/(` + charPat + `|[\*\?])+`
var globRe = regexp.MustCompile(`^/$|^((` + globRePart + `)+\|)*(` + globRePart + `)+$`)

// Supports unix/ruby-style glob patterns:
//  - `?` matches a single char in a single path component
//  - `*` matches zero or more chars in a single path component
//  - `**` matches zero or more chars in zero or more components
//  - `|` allows for alternate paths to be matched
func translateGlob(pat string) (string, error) {
	if !globRe.MatchString(pat) {
		return "", GlobError(pat)
	}

	outs := make([]string, len(pat))
	groupPattern := false
	i, double := 0, false
	for _, c := range pat {
		switch c {
		case '|':
			groupPattern = true
			fallthrough
		default:
			outs[i] = string(c)
			double = false
		case '.', '+', '-', '^', '$', '[', ']', '(', ')':
			outs[i] = `\` + string(c)
			double = false
		case '?':
			outs[i] = `[^/]`
			double = false
		case '*':
			if double {
				outs[i-1] = `.*`
			} else {
				outs[i] = `[^/]*`
			}
			double = !double
		}
		i++
	}
	outs = outs[0:i]
	outPat := strings.Join(outs, "")
	if groupPattern {
		/* We have to group the entire pattern when using alternation because
		 * otherwise the pipe matches a literal pipe */
		outPat = "(" + outPat + ")"
	}

	return "^" + outPat + "$", nil
}

// CompileGlob translates pat into a form more convenient for
// matching against paths in the store.
func CompileGlob(pat string) (*Glob, error) {
	s, err := translateGlob(pat)
	if err != nil {
		return nil, err
	}

	r, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}

	return &Glob{pat, s, r}, nil
}

// MustCompileGlob is like CompileGlob, but it panics if an error occurs,
// simplifying safe initialization of global variables holding glob patterns.
func MustCompileGlob(pat string) *Glob {
	g, err := CompileGlob(pat)
	if err != nil {
		panic(err)
	}
	return g
}

func (g *Glob) Match(path string) bool {
	return g.r.MatchString(path)
}

type GlobError string

func (e GlobError) Error() string {
	return "invalid glob pattern: " + string(e)
}

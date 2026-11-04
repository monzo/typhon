package typhon

import (
	"strings"
	"unicode/utf8"
)

// routeMatcher matches a path against a router pattern without using a regular expression. It is only built for
// patterns where it is guaranteed to behave exactly like the regular expression produced by Router.compile; any
// pattern outside that subset keeps using the regular expression.
//
// The supported subset is:
//   - literal text
//   - :name parameters that are followed by a '/' or by the end of the pattern
//   - a *residual (named or bare) as the very last component of the pattern
//
// In this subset every pattern has at most one way to match a path, so a single left-to-right scan gives the same
// result (and the same captured values) as the regular expression.
type routeMatcher struct {
	parts []routePart
}

type routePartKind uint8

const (
	routePartLiteral routePartKind = iota
	routePartParam
	routePartResidual
)

type routePart struct {
	kind routePartKind
	// lit is the literal text for routePartLiteral parts
	lit string
	// name is the capture name for routePartParam and named routePartResidual parts; it is empty for a bare residual
	name string
}

// newRouteMatcher returns a matcher for the pattern, or nil if the pattern falls outside the subset where a matcher is
// known to be equivalent to the compiled regular expression.
func newRouteMatcher(pattern string) *routeMatcher {
	// The regular expression engine treats invalid UTF-8 as U+FFFD, which a byte comparison would not. Leave these
	// unusual patterns to the regular expression.
	if !utf8.ValidString(pattern) || strings.ContainsRune(pattern, utf8.RuneError) {
		return nil
	}

	m := &routeMatcher{}
	pos := 0
	for _, loc := range routerComponentsRe.FindAllStringSubmatchIndex(pattern, -1) {
		if head := pattern[pos:loc[2]]; head != "" {
			m.parts = append(m.parts, routePart{kind: routePartLiteral, lit: head})
		}
		token := pattern[loc[2]:loc[3]]
		pos = loc[3]
		tail := pattern[pos:]
		switch sigil, name := token[0], token[1:]; sigil {
		case ':':
			// [^/]+ is only unambiguous if what follows can't be consumed by it
			if tail != "" && tail[0] != '/' {
				return nil
			}
			m.parts = append(m.parts, routePart{kind: routePartParam, name: name})
		case '*':
			// .*? followed by more pattern needs backtracking to find the shortest match; leave that to the regexp
			if tail != "" {
				return nil
			}
			m.parts = append(m.parts, routePart{kind: routePartResidual, name: name})
		default:
			return nil
		}
	}
	if tail := pattern[pos:]; tail != "" {
		m.parts = append(m.parts, routePart{kind: routePartLiteral, lit: tail})
	}
	return m
}

// match reports whether path matches the pattern. If it does and params is non-nil, captured parameters are written
// into params. params is not modified when there is no match.
func (m *routeMatcher) match(path string, params map[string]string) bool {
	if !m.scan(path, nil) {
		return false
	}
	if params != nil {
		m.scan(path, params)
	}
	return true
}

func (m *routeMatcher) scan(path string, params map[string]string) bool {
	i := 0
	for _, p := range m.parts {
		switch p.kind {
		case routePartLiteral:
			if !strings.HasPrefix(path[i:], p.lit) {
				return false
			}
			i += len(p.lit)
		case routePartParam:
			n := strings.IndexByte(path[i:], '/')
			if n < 0 {
				n = len(path) - i
			}
			if n == 0 { // [^/]+ needs at least one character
				return false
			}
			if params != nil {
				params[p.name] = path[i : i+n]
			}
			i += n
		case routePartResidual:
			// The regexp uses .*? for residuals, and . does not match a newline
			if strings.IndexByte(path[i:], '\n') >= 0 {
				return false
			}
			if params != nil && p.name != "" {
				params[p.name] = path[i:]
			}
			i = len(path)
		}
	}
	return i == len(path)
}

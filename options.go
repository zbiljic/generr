package main

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Option keys understood in a "+generr" annotation.
const (
	optFormat     = "format"
	optNameSuffix = "namesuffix"
	optParams     = "params"
	optSkip       = "skip"
)

// errorType is the parameter type that gets the default name "err" and is
// rendered with %w in a derived format.
const errorType = "error"

// option is a single key=value pair of a "+generr" annotation.
type option struct {
	key   string
	value string
}

// param is one parameter of a generated error function.
type param struct {
	name string
	typ  string
}

// spec describes one error function to generate. It is the parsed form of a
// single "+generr" annotation line.
type spec struct {
	format     string  // Format passed to fmt.Errorf; never empty once parsed.
	nameSuffix string  // Suffix of the generated function name; may be empty.
	params     []param // Parameters of the generated function, in order.
	args       []param // Parameters in the order they are passed to fmt.Errorf.
}

// parseSpec parses the options of a single "+generr" annotation line. The
// empty string yields the default error function: one "err error" parameter
// wrapped as "%s: %w".
func parseSpec(s string) (spec, error) {
	opts, err := splitOptions(s)
	if err != nil {
		return spec{}, err
	}

	sp := spec{params: []param{{name: "err", typ: errorType}}}

	for _, o := range opts {
		switch o.key {
		case optFormat:
			if o.value == "" {
				return spec{}, errors.New("option format: value must not be empty")
			}
			sp.format = o.value
		case optNameSuffix:
			if o.value == "" {
				return spec{}, errors.New("option namesuffix: value must not be empty")
			}
			if !token.IsIdentifier("err" + o.value) {
				return spec{}, fmt.Errorf("option namesuffix: %q cannot be part of a function name", o.value)
			}
			sp.nameSuffix = titleCase(o.value)
		case optParams:
			sp.params, err = parseParams(o.value)
			if err != nil {
				return spec{}, fmt.Errorf("option params: %w", err)
			}
		default:
			return spec{}, fmt.Errorf("unknown option %q (valid options: %s, %s, %s)",
				o.key, optFormat, optNameSuffix, optParams)
		}
	}

	if sp.format == "" {
		// No format given: derive one, which also decides the argument order.
		sp.format, sp.args = defaultFormat(sp.params)
		return sp, nil
	}

	sp.args = sp.params
	if err := validateFormat(sp.format, sp.args); err != nil {
		return spec{}, err
	}

	return sp, nil
}

// splitOptions splits a comma-separated list of key=value pairs. A value is
// either taken verbatim up to the next comma, or, when it starts with a double
// quote, is a Go string literal, which lets it contain commas.
func splitOptions(s string) ([]option, error) {
	var opts []option
	seen := make(map[string]bool)

	for rest := strings.TrimSpace(s); rest != ""; rest = strings.TrimSpace(rest) {
		key, after, ok := strings.Cut(rest, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.Contains(key, ",") {
			word, _, _ := strings.Cut(rest, ",")
			word = strings.TrimSpace(word)
			if word == optSkip {
				return nil, errors.New(`"skip" cannot be combined with other options or other +generr lines`)
			}
			return nil, fmt.Errorf("invalid option %q: expected key=value", word)
		}
		if seen[key] {
			return nil, fmt.Errorf("option %s is set more than once", key)
		}
		seen[key] = true

		var value string
		if strings.HasPrefix(after, `"`) {
			quoted, err := strconv.QuotedPrefix(after)
			if err != nil {
				return nil, fmt.Errorf("option %s: invalid quoted value %s", key, after)
			}
			value, err = strconv.Unquote(quoted)
			if err != nil {
				return nil, fmt.Errorf("option %s: invalid quoted value %s", key, quoted)
			}
			rest = strings.TrimSpace(after[len(quoted):])
			if rest != "" && !strings.HasPrefix(rest, ",") {
				return nil, fmt.Errorf("option %s: unexpected text %q after quoted value", key, rest)
			}
			rest = strings.TrimPrefix(rest, ",")
		} else {
			value, rest, _ = strings.Cut(after, ",")
		}

		opts = append(opts, option{key: key, value: value})
	}

	return opts, nil
}

// parseParams parses a semicolon-separated parameter list. Each parameter is
// a type, optionally followed by its name in parentheses: "string(id);error".
// Unnamed parameters are called "err" when their type is error, and "p<index>"
// otherwise.
func parseParams(s string) ([]param, error) {
	var params []param
	seen := make(map[string]bool)

	for i, p := range strings.Split(s, ";") {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("parameter %d is empty", i+1)
		}

		prm := param{name: "p" + strconv.Itoa(i), typ: p}
		if p == errorType {
			prm.name = "err"
		}

		// A trailing "(name)" names the parameter. Anything else in parentheses,
		// like the "()" of a func type, belongs to the type.
		if idx := strings.LastIndex(p, "("); idx > 0 && strings.HasSuffix(p, ")") {
			if name := p[idx+1 : len(p)-1]; token.IsIdentifier(name) {
				prm.name = name
				prm.typ = strings.TrimSpace(p[:idx])
			}
		}

		if seen[prm.name] {
			return nil, fmt.Errorf("parameter name %q is used more than once; name them as type(name)", prm.name)
		}
		seen[prm.name] = true

		params = append(params, prm)
	}

	return params, nil
}

// defaultFormat derives the format for a parameter list that has no explicit
// one. The operation name comes first, then the non-error parameters in
// parentheses, then the wrapped errors: "%s(%v/%v): %w". It also returns the
// parameters in the order the format consumes them.
func defaultFormat(params []param) (string, []param) {
	var values, errs []param
	for _, p := range params {
		if p.typ == errorType {
			errs = append(errs, p)
		} else {
			values = append(values, p)
		}
	}

	var b strings.Builder
	b.WriteString("%s")
	if len(values) > 0 {
		b.WriteString("(")
		b.WriteString(strings.Repeat("%v/", len(values)-1))
		b.WriteString("%v)")
	}
	b.WriteString(strings.Repeat(": %w", len(errs)))

	return b.String(), append(values, errs...)
}

// validateFormat checks that format consumes exactly the operation name
// followed by args. Formats that use explicit argument indexes or '*' are not
// checked, because their verbs do not map one-to-one onto the arguments.
func validateFormat(format string, args []param) error {
	verbs, ok := formatVerbs(format)
	if !ok {
		return nil
	}

	if want := 1 + len(args); len(verbs) != want {
		return fmt.Errorf("format %q has %d verbs but gets %d arguments: the operation name, then %s",
			format, len(verbs), want, describeParams(args))
	}

	if !strings.ContainsRune("svq", verbs[0]) {
		return fmt.Errorf("format %q must start by consuming the operation name with %%s, not %%%c",
			format, verbs[0])
	}

	for i, p := range args {
		if verbs[i+1] == 'w' && isBasicType(p.typ) {
			return fmt.Errorf("format %q applies %%w to parameter %s of type %s, which is not an error",
				format, p.name, p.typ)
		}
	}

	return nil
}

// formatVerbs returns the verbs of a fmt format string, in order. It reports
// false when the verbs cannot be matched one-to-one with arguments: explicit
// argument indexes, '*' widths or precisions, or a dangling '%'.
func formatVerbs(format string) ([]rune, bool) {
	var verbs []rune

	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++

		// Skip flags, width and precision.
		for i < len(format) && strings.IndexByte("+-# 0123456789.", format[i]) >= 0 {
			i++
		}
		if i >= len(format) {
			return nil, false
		}

		switch format[i] {
		case '*', '[':
			return nil, false
		case '%':
			continue
		}

		verb, size := utf8.DecodeRuneInString(format[i:])
		verbs = append(verbs, verb)
		i += size - 1
	}

	return verbs, true
}

// isBasicType reports whether typ names a predeclared basic type, such as
// string or int. Those can never be an error.
func isBasicType(typ string) bool {
	tn, ok := types.Universe.Lookup(typ).(*types.TypeName)
	if !ok {
		return false
	}
	_, ok = tn.Type().(*types.Basic)
	return ok
}

// describeParams renders a parameter list for error messages.
func describeParams(params []param) string {
	if len(params) == 0 {
		return "no parameters"
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.name + " " + p.typ
	}
	return strings.Join(parts, ", ")
}

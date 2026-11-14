package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// generateTestdata runs the generator for the Service type of a testdata
// package.
func generateTestdata(t *testing.T, pkg string, exportedOnly bool) ([]byte, error) {
	t.Helper()

	g := Generator{
		trimPrefix:   "testdata",
		exportedOnly: exportedOnly,
		logf:         t.Logf,
	}
	if err := g.parsePackage([]string{"./testdata/" + pkg}, nil); err != nil {
		t.Fatalf("parsePackage: %v", err)
	}

	return g.run([]string{"Service"})
}

// TestGolden compares the generated file with the one committed in testdata.
// That file is part of the testdata package, so vetting the package also vets
// the generated code.
func TestGolden(t *testing.T) {
	got, err := generateTestdata(t, "svc", false)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	golden := filepath.Join("testdata", "svc", "service_generr.go")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run 'go test -update' to create it)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("generated code differs from %s (run 'go test -update' to accept):\n%s", golden, got)
	}

	out, err := exec.Command("go", "vet", "./testdata/svc").CombinedOutput()
	if err != nil {
		t.Errorf("go vet rejects the generated code: %v\n%s", err, out)
	}
}

func TestExportedOnly(t *testing.T) {
	got, err := generateTestdata(t, "svc", true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if bytes.Contains(got, []byte("errUnexported")) {
		t.Errorf("generated an error function for an unexported method:\n%s", got)
	}
	if !bytes.Contains(got, []byte("errDefault")) {
		t.Errorf("missing the error function of an exported method:\n%s", got)
	}
}

// TestRejectedAnnotation checks that a bad annotation fails the whole run,
// naming the method, instead of producing a file.
func TestRejectedAnnotation(t *testing.T) {
	got, err := generateTestdata(t, "bad", false)
	if err == nil {
		t.Fatalf("run succeeded, want an error; output:\n%s", got)
	}
	for _, want := range []string{"Service.Broken", "has 1 verbs but gets 2 arguments"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestUnknownType(t *testing.T) {
	g := Generator{logf: t.Logf}
	if err := g.parsePackage([]string{"./testdata/svc"}, nil); err != nil {
		t.Fatalf("parsePackage: %v", err)
	}
	if _, err := g.run([]string{"Missing"}); err == nil {
		t.Error("run succeeded for a type without methods, want an error")
	}
}

func TestParseSpec(t *testing.T) {
	errParam := param{name: "err", typ: "error"}
	id := param{name: "id", typ: "string"}

	tests := []struct {
		name string
		in   string
		want spec
	}{
		{
			name: "empty",
			in:   "",
			want: spec{format: "%s: %w", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "format",
			in:   "format=%s: failed: %w",
			want: spec{format: "%s: failed: %w", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "format without wrapping",
			in:   "format=%s: failed: %v",
			want: spec{format: "%s: failed: %v", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "namesuffix is title-cased",
			in:   "namesuffix=forID",
			want: spec{format: "%s: %w", nameSuffix: "ForID", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "params derive the format",
			in:   "params=string(id);error",
			want: spec{format: "%s(%v): %w", params: []param{id, errParam}, args: []param{id, errParam}},
		},
		{
			name: "derived format moves errors last",
			in:   "params=error;string(id);int",
			want: spec{
				format: "%s(%v/%v): %w",
				params: []param{errParam, id, {name: "p2", typ: "int"}},
				args:   []param{id, {name: "p2", typ: "int"}, errParam},
			},
		},
		{
			name: "params without error",
			in:   "params=string(id)",
			want: spec{format: "%s(%v)", params: []param{id}, args: []param{id}},
		},
		{
			name: "only an error param",
			in:   "params=error",
			want: spec{format: "%s: %w", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "explicit format keeps the declared order",
			in:   "format=%s(%s): %w,params=string(id);error,namesuffix=forID",
			want: spec{format: "%s(%s): %w", nameSuffix: "ForID", params: []param{id, errParam}, args: []param{id, errParam}},
		},
		{
			name: "format with punctuation",
			in:   "format=%s(owner:'%s'/after:'%s'): %w,params=string(a);string(b);error",
			want: spec{
				format: "%s(owner:'%s'/after:'%s'): %w",
				params: []param{{name: "a", typ: "string"}, {name: "b", typ: "string"}, errParam},
				args:   []param{{name: "a", typ: "string"}, {name: "b", typ: "string"}, errParam},
			},
		},
		{
			name: "quoted format with comma and equal sign",
			in:   `format="%s: id=%d, n=%d: %w",params=int(id);int(n);error`,
			want: spec{
				format: "%s: id=%d, n=%d: %w",
				params: []param{{name: "id", typ: "int"}, {name: "n", typ: "int"}, errParam},
				args:   []param{{name: "id", typ: "int"}, {name: "n", typ: "int"}, errParam},
			},
		},
		{
			name: "quoted params and escaped quote",
			in:   `params="string(id);error", format="%s(\"%s\"): %w"`,
			want: spec{format: `%s("%s"): %w`, params: []param{id, errParam}, args: []param{id, errParam}},
		},
		{
			name: "unquoted value keeps an equal sign",
			in:   "format=%s: id=%d: %w,params=int(id);error",
			want: spec{
				format: "%s: id=%d: %w",
				params: []param{{name: "id", typ: "int"}, errParam},
				args:   []param{{name: "id", typ: "int"}, errParam},
			},
		},
		{
			name: "surrounding whitespace",
			in:   "  format=%s: failed: %w  ",
			want: spec{format: "%s: failed: %w", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "composite and func types",
			in:   "format=%s: %v %v %v %v,params=[]string(ids);map[string]int;func();*Thing(t)",
			want: spec{
				format: "%s: %v %v %v %v",
				params: []param{
					{name: "ids", typ: "[]string"},
					{name: "p1", typ: "map[string]int"},
					{name: "p2", typ: "func()"},
					{name: "t", typ: "*Thing"},
				},
				args: []param{
					{name: "ids", typ: "[]string"},
					{name: "p1", typ: "map[string]int"},
					{name: "p2", typ: "func()"},
					{name: "t", typ: "*Thing"},
				},
			},
		},
		{
			name: "percent literal is not a verb",
			in:   "format=%s: 100%% failed: %w",
			want: spec{format: "%s: 100%% failed: %w", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "flags and width",
			in:   "format=%-10s: %+5.2f: %w,params=float64(ratio);error",
			want: spec{
				format: "%-10s: %+5.2f: %w",
				params: []param{{name: "ratio", typ: "float64"}, errParam},
				args:   []param{{name: "ratio", typ: "float64"}, errParam},
			},
		},
		{
			name: "argument indexes are not validated",
			in:   "format=%[2]v in %[1]s",
			want: spec{format: "%[2]v in %[1]s", params: []param{errParam}, args: []param{errParam}},
		},
		{
			name: "custom error type may be wrapped",
			in:   "format=%s: %w,params=*MyError(cause)",
			want: spec{
				format: "%s: %w",
				params: []param{{name: "cause", typ: "*MyError"}},
				args:   []param{{name: "cause", typ: "*MyError"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSpec(tt.in)
			if err != nil {
				t.Fatalf("parseSpec(%q): %v", tt.in, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSpec(%q)\n got: %+v\nwant: %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSpecErrors(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{"format drops the operation name", "format=failed: %w", "has 1 verbs but gets 2 arguments"},
		{"format with too many verbs", "format=%s: %s: %w", "has 3 verbs but gets 2 arguments"},
		{"format ignores a param", "format=%s: %w,params=string(id);error", "has 2 verbs but gets 3 arguments"},
		{"format without verbs", "format=failed", "has 0 verbs but gets 2 arguments"},
		{"operation name is not printable", "format=%d: %w", "must start by consuming the operation name"},
		{"operation name wrapped", "format=%w: %s", "must start by consuming the operation name"},
		{"wraps a string", "format=%s: %w (%s),params=string(id);error", "applies %w to parameter id of type string"},
		{"empty format", "format=", "option format: value must not be empty"},
		{"empty namesuffix", "namesuffix=", "option namesuffix: value must not be empty"},
		{"namesuffix is not an identifier", "namesuffix=not-found", "cannot be part of a function name"},
		{"unknown option", "nameSuffix=Foo", `unknown option "nameSuffix"`},
		{"missing value", "format", `invalid option "format": expected key=value`},
		{"missing key", "=x", "expected key=value"},
		{"comma in unquoted format", "format=%s: a, b: %w", `invalid option "b: %w": expected key=value`},
		{"duplicate option", "namesuffix=A,namesuffix=B", "option namesuffix is set more than once"},
		{"skip with options", "skip,format=%s: %w", `"skip" cannot be combined`},
		{"options with skip", "format=%s: %w,skip", `"skip" cannot be combined`},
		{"unterminated quote", `format="%s: %w`, "invalid quoted value"},
		{"text after quote", `format="%s: %w"x`, "unexpected text"},
		{"empty param", "params=string(id);;error", "parameter 2 is empty"},
		{"empty params", "params=", "parameter 1 is empty"},
		{"duplicate param names", "params=error;error", `parameter name "err" is used more than once`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSpec(tt.in)
			if err == nil {
				t.Fatalf("parseSpec(%q) = %+v, want an error containing %q", tt.in, got, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseSpec(%q) error = %q, want it to contain %q", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestFormatVerbs(t *testing.T) {
	tests := []struct {
		format string
		want   string
		ok     bool
	}{
		{"", "", true},
		{"no verbs", "", true},
		{"%s: %w", "sw", true},
		{"%s(%v/%v): %w", "svvw", true},
		{"100%%", "", true},
		{"%+v %#v %05d %6.2f %x", "vvdfx", true},
		{"%[1]s", "", false},
		{"%*d", "", false},
		{"%.*f", "", false},
		{"trailing %", "", false},
	}

	for _, tt := range tests {
		got, ok := formatVerbs(tt.format)
		if string(got) != tt.want || ok != tt.ok {
			t.Errorf("formatVerbs(%q) = %q, %v; want %q, %v", tt.format, string(got), ok, tt.want, tt.ok)
		}
	}
}

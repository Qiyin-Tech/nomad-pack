package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/nomad-pack/sdk/render"
)

func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"metadata.hcl":             "app { url = \"\" }\npack {\n name = \"example\"\n version = \"1.0.0\"\n}\n",
		"variables.hcl":            "variable \"count\" { type = number }\nvariable \"settings\" {\n type = object({ enabled = optional(bool, true) })\n default = {}\n}\n",
		"templates/main.nomad.tpl": "job \"example\" {\n count = [[ var \"count\" . ]]\n meta { enabled = [[ var \"settings.enabled\" . | quote ]] }\n}\n",
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestVariableContract(t *testing.T) {
	t.Setenv("NOMAD_PACK_VAR_count", "must-not-be-read")
	dir := fixture(t)
	for _, tc := range []struct {
		name, input string
		warnings    int
		fails       bool
	}{
		{"known", "count = 2", 0, false},
		{"future", "count = 2\nfuture = \"PRIVATE_VALUE\"", 1, false},
		{"future_pack", "count = 2\nfuture.option = \"PRIVATE_VALUE\"", 1, false},
		{"mixed", "future = true\ncount = \"not-a-number\"", 1, true},
		{"future_required", "future = true", 1, true},
		{"wrong_type", "count = \"not-a-number\"", 0, true},
		{"nested_type", "count = 2\nsettings = { enabled = [] }", 0, true},
		{"optional", "count = 2\nsettings = {}", 0, false},
		{"required", "", 0, true},
		{"syntax", "count =", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vars := filepath.Join(t.TempDir(), "input.hcl")
			if err := os.WriteFile(vars, []byte(tc.input), 0600); err != nil {
				t.Fatal(err)
			}
			out, err := render.Render(render.Options{Path: dir, VariableFiles: []string{vars}})
			if (err != nil) != tc.fails {
				t.Fatalf("expected failure=%v, got %v", tc.fails, err)
			}
			if tc.fails {
				if !out.Diagnostics.HasErrors() || strings.Contains(err.Error(), "尚未声明") {
					t.Fatalf("warning/error diagnostics mixed or lost: %v; %v", err, out.Diagnostics)
				}
				return
			}
			if len(out.Templates) != 1 || len(out.Diagnostics) != tc.warnings {
				t.Fatalf("unexpected result: %+v", out)
			}
			for _, content := range out.Templates {
				if !strings.Contains(content, "count = 2") {
					t.Fatalf("override not rendered: %s", content)
				}
			}
			for _, diag := range out.Diagnostics {
				if diag.Subject == nil || strings.Contains(diag.Error(), "PRIVATE_VALUE") {
					t.Fatalf("unsafe warning: %v", diag)
				}
			}
		})
	}
}

func TestSources(t *testing.T) {
	dir := fixture(t)
	vars := filepath.Join(t.TempDir(), "input.hcl")
	if err := os.WriteFile(vars, []byte("count = 2"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cli   map[string]string
		count string
		files []string
	}{
		{nil, "1", nil}, {nil, "2", []string{vars}}, {map[string]string{"count": "3"}, "3", []string{vars}},
	} {
		out, err := render.Render(render.Options{Path: dir, VariableFiles: tc.files,
			Environment: map[string]string{"count": "1"}, Variables: tc.cli})
		if err != nil {
			t.Fatal(err)
		}
		for _, content := range out.Templates {
			if !strings.Contains(content, "count = "+tc.count) || !strings.Contains(content, `enabled = "true"`) {
				t.Fatalf("bad override precedence or defaults: %s", content)
			}
		}
	}
}

func TestTemplateErrorLocation(t *testing.T) {
	dir := fixture(t)
	if err := os.WriteFile(filepath.Join(dir, "templates/main.nomad.tpl"), []byte(`[[ fail "broken" ]]`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := render.Render(render.Options{Path: dir, Variables: map[string]string{"count": "1"}})
	if err == nil || !strings.Contains(err.Error(), "main.nomad.tpl") || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("template location lost: %v", err)
	}
}

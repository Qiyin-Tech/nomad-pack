package format_test

import (
	"bytes"
	"strings"
	"testing"

	packfmt "github.com/hashicorp/nomad-pack/sdk/format"
)

func TestTemplatePreservation(t *testing.T) {
	for _, action := range []string{
		`[[ var "name" . | quote ]]`,
		`[[ printf "%s ]]" "x" | quote ]]`,
		"[[\n var \"name\" .\n]]",
		"[[- /* comment ]]\n */ -]]",
		"[[ printf `]]` ]]",
	} {
		src := []byte("job " + action + " {\ncount=1\n# __NOMAD_PACK_TPL_0__\n}\n")
		out, err := packfmt.Source("job.nomad.tpl", src)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), action) || !strings.Contains(string(out), "  count = 1") || !strings.Contains(string(out), "# __NOMAD_PACK_TPL_0__") {
			t.Fatalf("corrupt format: %s", out)
		}
		second, err := packfmt.Source("job.nomad.tpl", out)
		if err != nil || !bytes.Equal(out, second) {
			t.Fatalf("not idempotent: %s, %v", second, err)
		}
	}
	for name, src := range map[string]string{"bad.tpl": "[[ \"open", "comment.tpl": "[[ /*", "action.tpl": "[[ var", "bad.hcl": "job {", "other.txt": ""} {
		if _, err := packfmt.Source(name, []byte(src)); err == nil {
			t.Fatalf("accepted malformed input %s", name)
		}
	}
}

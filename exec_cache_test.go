package pongo2_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/rudderlabs/pongo2/v6"
)

func resetExecTemplateCache(t *testing.T, capacity, maxBodyBytes int) {
	t.Helper()
	pongo2.SetExecTemplateCache(capacity, maxBodyBytes)
	t.Cleanup(func() {
		pongo2.SetExecTemplateCache(16384, 1024)
	})
}

func execTemplateOutput(t *testing.T, tpl *pongo2.Template, ctx pongo2.Context) string {
	t.Helper()
	out, err := tpl.Execute(ctx)
	if err != nil {
		t.Fatalf("execute template: %v", err)
	}
	return out
}

func newExecTemplate(t *testing.T, source string) *pongo2.Template {
	t.Helper()
	set := pongo2.NewSet(t.Name(), pongo2.MustNewLocalFileSystemLoader(""))
	tpl, err := set.FromString(source)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	return tpl
}

func TestExecTemplateCacheHitsOnSecondExecution(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	tpl := newExecTemplate(t, `{% exec %}{{ body }}{% endexec %}`)
	ctx := pongo2.Context{"body": "Hello {{ name }}", "name": "Ada"}

	if got := execTemplateOutput(t, tpl, ctx); got != "Hello Ada" {
		t.Fatalf("first output = %q, want %q", got, "Hello Ada")
	}
	if got := execTemplateOutput(t, tpl, ctx); got != "Hello Ada" {
		t.Fatalf("second output = %q, want %q", got, "Hello Ada")
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 1 || misses != 1 || evictions != 0 || entries != 1 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=1 misses=1 evictions=0 entries=1", hits, misses, evictions, entries)
	}
}

func TestExecTemplateCacheSkipsOversizedBodies(t *testing.T) {
	resetExecTemplateCache(t, 16, 8)

	tpl := newExecTemplate(t, `{% exec %}{{ body }}{% endexec %}`)
	ctx := pongo2.Context{"body": "Hello {{ name }}", "name": "Ada"}

	if got := execTemplateOutput(t, tpl, ctx); got != "Hello Ada" {
		t.Fatalf("first output = %q, want %q", got, "Hello Ada")
	}
	if got := execTemplateOutput(t, tpl, ctx); got != "Hello Ada" {
		t.Fatalf("second output = %q, want %q", got, "Hello Ada")
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 0 || misses != 2 || evictions != 0 || entries != 0 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=2 evictions=0 entries=0", hits, misses, evictions, entries)
	}
}

func TestExecTemplateCacheEvictsLeastRecentlyUsed(t *testing.T) {
	resetExecTemplateCache(t, 2, 1024)

	tpl := newExecTemplate(t, `{% exec %}{{ body }}{% endexec %}`)
	for i := 0; i < 3; i++ {
		want := fmt.Sprintf("body-%d", i)
		if got := execTemplateOutput(t, tpl, pongo2.Context{"body": want}); got != want {
			t.Fatalf("output %d = %q, want %q", i, got, want)
		}
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 0 || misses != 3 || evictions != 1 || entries != 2 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=3 evictions=1 entries=2", hits, misses, evictions, entries)
	}
}

func TestExecTemplateCacheCanBeDisabled(t *testing.T) {
	resetExecTemplateCache(t, 0, 1024)

	tpl := newExecTemplate(t, `{% exec %}{{ body }}{% endexec %}`)
	ctx := pongo2.Context{"body": "Hello {{ name }}", "name": "Ada"}

	for i := 0; i < 3; i++ {
		if got := execTemplateOutput(t, tpl, ctx); got != "Hello Ada" {
			t.Fatalf("output %d = %q, want %q", i, got, "Hello Ada")
		}
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 0 || misses != 3 || evictions != 0 || entries != 0 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=3 evictions=0 entries=0", hits, misses, evictions, entries)
	}
}

func TestExecTemplateCacheConcurrentExecution(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	tpl := newExecTemplate(t, `{% exec %}{{ body }}{% endexec %}`)
	const goroutines = 16

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := pongo2.Context{
				"body":   "worker={{ worker }}",
				"worker": i,
			}
			want := fmt.Sprintf("worker=%d", i)
			for j := 0; j < 100; j++ {
				if got := execTemplateOutput(t, tpl, ctx); got != want {
					t.Errorf("worker %d output %d = %q, want %q", i, j, got, want)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	hits, misses, _, entries := pongo2.ExecTemplateCacheStats()
	if entries != 1 {
		t.Fatalf("entries = %d, want 1", entries)
	}
	if misses < 1 || misses > goroutines {
		t.Fatalf("misses = %d, want between 1 and %d", misses, goroutines)
	}
	if hits == 0 {
		t.Fatalf("hits = 0, want concurrent cache hits")
	}
}

func TestAllowMissingValUsesExecTemplateCache(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	tpl := newExecTemplate(t, `{% allowmissingval %}{{ missing }}{% endallowmissingval %}`)
	if got := execTemplateOutput(t, tpl, nil); got != "" {
		t.Fatalf("first output = %q, want empty string", got)
	}
	if got := execTemplateOutput(t, tpl, nil); got != "" {
		t.Fatalf("second output = %q, want empty string", got)
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 1 || misses != 1 || evictions != 0 || entries != 1 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=1 misses=1 evictions=0 entries=1", hits, misses, evictions, entries)
	}
}

func TestExecTemplateCacheBypassesStatefulTags(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	tests := []struct {
		name string
		body string
		ctx  pongo2.Context
		want string
	}{
		{
			name: "cycle",
			body: `{% cycle first second %}`,
			ctx:  pongo2.Context{"first": "A", "second": "B"},
			want: "A",
		},
		{
			name: "ifchanged values",
			body: `{% ifchanged secret %}changed{% else %}same{% endifchanged %}`,
			ctx:  pongo2.Context{"secret": "same-across-executions"},
			want: "changed",
		},
		{
			name: "ifchanged content",
			body: `{% ifchanged %}same{% endifchanged %}`,
			want: "same",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExecTemplateCache(t, 16, 1024)

			tpl := newExecTemplate(t, `{% exec %}{{ body|safe }}{% endexec %}`)
			ctx := pongo2.Context{"body": tt.body}
			ctx.Update(tt.ctx)

			for i := 0; i < 3; i++ {
				if got := execTemplateOutput(t, tpl, ctx); got != tt.want {
					t.Fatalf("output %d = %q, want %q", i, got, tt.want)
				}
			}

			hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
			if hits != 0 || misses != 3 || evictions != 0 || entries != 0 {
				t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=3 evictions=0 entries=0", hits, misses, evictions, entries)
			}
		})
	}
}

func TestAllowMissingValCacheBypassesStatefulTags(t *testing.T) {
	tests := []struct {
		name string
		body string
		ctx  pongo2.Context
		want string
	}{
		{
			name: "cycle",
			body: `{% cycle first second %}`,
			ctx:  pongo2.Context{"first": "A", "second": "B"},
			want: "A",
		},
		{
			name: "ifchanged values",
			body: `{% ifchanged secret %}changed{% else %}same{% endifchanged %}`,
			ctx:  pongo2.Context{"secret": "same-across-executions"},
			want: "changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExecTemplateCache(t, 16, 1024)

			tpl := newExecTemplate(t, `{% allowmissingval %}{{ body|safe }}{% endallowmissingval %}`)
			ctx := pongo2.Context{"body": tt.body}
			ctx.Update(tt.ctx)

			for i := 0; i < 3; i++ {
				if got := execTemplateOutput(t, tpl, ctx); got != tt.want {
					t.Fatalf("output %d = %q, want %q", i, got, tt.want)
				}
			}

			hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
			if hits != 0 || misses != 3 || evictions != 0 || entries != 0 {
				t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=3 evictions=0 entries=0", hits, misses, evictions, entries)
			}
		})
	}
}

func TestExecTemplateCacheStatefulTagsConcurrentBypass(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	tpl := newExecTemplate(t, `{% exec %}{{ body|safe }}{% endexec %}`)
	const goroutines = 16

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := pongo2.Context{
				"body":   `{% cycle first second %}:{% ifchanged secret %}changed{% else %}same{% endifchanged %}`,
				"first":  "A",
				"second": "B",
				"secret": i,
			}
			for j := 0; j < 100; j++ {
				if got := execTemplateOutput(t, tpl, ctx); got != "A:changed" {
					t.Errorf("worker %d output %d = %q, want %q", i, j, got, "A:changed")
					return
				}
			}
		}(i)
	}
	wg.Wait()

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 0 || misses != goroutines*100 || evictions != 0 || entries != 0 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=%d evictions=0 entries=0", hits, misses, evictions, entries, goroutines*100)
	}
}

func TestExecTemplateCacheBypassesStaticLoaderDependentTags(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	dir := t.TempDir()
	includePath := dir + string(os.PathSeparator) + "fragment.tpl"
	if err := os.WriteFile(includePath, []byte("before"), 0o600); err != nil {
		t.Fatalf("write initial include: %v", err)
	}

	set := pongo2.NewSet("include", pongo2.MustNewLocalFileSystemLoader(dir))
	tpl := pongo2.Must(set.FromString(`{% exec %}{% verbatim %}{% include "fragment.tpl" %}{% endverbatim %}{% endexec %}`))
	if got := execTemplateOutput(t, tpl, nil); got != "before" {
		t.Fatalf("first output = %q, want %q", got, "before")
	}

	if err := os.WriteFile(includePath, []byte("after"), 0o600); err != nil {
		t.Fatalf("write updated include: %v", err)
	}
	if got := execTemplateOutput(t, tpl, nil); got != "after" {
		t.Fatalf("second output = %q, want %q", got, "after")
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 0 || misses != 2 || evictions != 0 || entries != 0 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=2 evictions=0 entries=0", hits, misses, evictions, entries)
	}
}

func TestExecTemplateCacheBypassesWhenTemplateSetDebugEnabled(t *testing.T) {
	resetExecTemplateCache(t, 16, 1024)

	set := pongo2.NewSet("debug", pongo2.MustNewLocalFileSystemLoader(""))
	set.Debug = true
	tpl := pongo2.Must(set.FromString(`{% exec %}{{ body }}{% endexec %}`))
	ctx := pongo2.Context{"body": "Hello {{ name }}", "name": "Ada"}

	for i := 0; i < 2; i++ {
		if got := execTemplateOutput(t, tpl, ctx); got != "Hello Ada" {
			t.Fatalf("output %d = %q, want %q", i, got, "Hello Ada")
		}
	}

	hits, misses, evictions, entries := pongo2.ExecTemplateCacheStats()
	if hits != 0 || misses != 2 || evictions != 0 || entries != 0 {
		t.Fatalf("stats = hits=%d misses=%d evictions=%d entries=%d, want hits=0 misses=2 evictions=0 entries=0", hits, misses, evictions, entries)
	}
}

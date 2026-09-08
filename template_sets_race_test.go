package pongo2_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/rudderlabs/pongo2/v6"
)

func TestTemplateSetFirstTemplateCreatedIsAtomic(t *testing.T) {
	set := pongo2.NewSet("first template race", pongo2.MustNewLocalFileSystemLoader(""))

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := set.FromString(fmt.Sprintf("hello {{ name }} %d", i)); err != nil {
				t.Errorf("FromString(%d): %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	if err := set.BanTag("if"); err == nil || !strings.Contains(err.Error(), "after you've added your first template") {
		t.Fatalf("BanTag after first template error = %v, want already-created error", err)
	}
	if err := set.BanFilter("escape"); err == nil || !strings.Contains(err.Error(), "after you've added your first template") {
		t.Fatalf("BanFilter after first template error = %v, want already-created error", err)
	}
}

func TestTemplateSetBanTagAndBanFilterBeforeFirstTemplate(t *testing.T) {
	set := pongo2.NewSet("sandbox before first template", pongo2.MustNewLocalFileSystemLoader(""))
	if err := set.BanTag("if"); err != nil {
		t.Fatalf("BanTag before first template error = %v", err)
	}
	if err := set.BanFilter("escape"); err != nil {
		t.Fatalf("BanFilter before first template error = %v", err)
	}
}

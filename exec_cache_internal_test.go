package pongo2

import "testing"

func resetExecTemplateCacheInternal(t *testing.T, capacity, maxBodyBytes int) {
	t.Helper()
	SetExecTemplateCache(capacity, maxBodyBytes)
	t.Cleanup(func() {
		SetExecTemplateCache(defaultExecTemplateCacheCapacity, defaultExecTemplateCacheMaxBodyBytes)
	})
}

func requireExecTemplateCacheStatsInternal(t *testing.T, wantHits, wantMisses, wantEvictions int64, wantEntries int) {
	t.Helper()
	hits, misses, evictions, entries := ExecTemplateCacheStats()
	if hits != wantHits || misses != wantMisses || evictions != wantEvictions || entries != wantEntries {
		t.Fatalf("ExecTemplateCacheStats() = hits=%d misses=%d evictions=%d entries=%d; want hits=%d misses=%d evictions=%d entries=%d",
			hits, misses, evictions, entries, wantHits, wantMisses, wantEvictions, wantEntries)
	}
}

func TestExecTemplateCacheDoesNotReuseAcrossTemplateSets(t *testing.T) {
	resetExecTemplateCacheInternal(t, 4, 1024)

	allowedSet := NewSet("allowed", MustNewLocalFileSystemLoader(""))
	body := []byte("{% for value in values %}{{ value }}{% endfor %}")
	if _, err := allowedSet.fromBytesCached(body); err != nil {
		t.Fatalf("allowedSet.fromBytesCached() error = %v", err)
	}

	bannedSet := NewSet("banned", MustNewLocalFileSystemLoader(""))
	if err := bannedSet.BanTag("for"); err != nil {
		t.Fatalf("BanTag() error = %v", err)
	}
	if _, err := bannedSet.fromBytesCached(body); err == nil {
		t.Fatal("bannedSet.fromBytesCached() succeeded; want banned tag error")
	}

	requireExecTemplateCacheStatsInternal(t, 0, 2, 0, 1)
}

func TestExecTemplateCacheKeysSameBodyByTemplateSet(t *testing.T) {
	resetExecTemplateCacheInternal(t, 2, 1024)

	body := []byte("Hello {{ name }}")
	setA := NewSet("set-a", MustNewLocalFileSystemLoader(""))
	setB := NewSet("set-b", MustNewLocalFileSystemLoader(""))
	setC := NewSet("set-c", MustNewLocalFileSystemLoader(""))

	if _, err := setA.fromBytesCached(body); err != nil {
		t.Fatalf("setA.fromBytesCached() error = %v", err)
	}
	if _, err := setB.fromBytesCached(body); err != nil {
		t.Fatalf("setB.fromBytesCached() error = %v", err)
	}
	requireExecTemplateCacheStatsInternal(t, 0, 2, 0, 2)

	if _, err := setA.fromBytesCached(body); err != nil {
		t.Fatalf("setA second fromBytesCached() error = %v", err)
	}
	requireExecTemplateCacheStatsInternal(t, 1, 2, 0, 2)

	if _, err := setC.fromBytesCached(body); err != nil {
		t.Fatalf("setC.fromBytesCached() error = %v", err)
	}
	requireExecTemplateCacheStatsInternal(t, 1, 3, 1, 2)
}

func TestExecTemplateCacheDoesNotReuseAfterTemplateSetOptionsChange(t *testing.T) {
	resetExecTemplateCacheInternal(t, 4, 1024)

	set := NewSet("options", MustNewLocalFileSystemLoader(""))
	body := []byte("{% if true %}\ntrimmed{% endif %}")
	tplWithoutTrim, err := set.fromBytesCached(body)
	if err != nil {
		t.Fatalf("first fromBytesCached() error = %v", err)
	}
	set.Options.TrimBlocks = true
	tplWithTrim, err := set.fromBytesCached(body)
	if err != nil {
		t.Fatalf("second fromBytesCached() error = %v", err)
	}
	if tplWithoutTrim == tplWithTrim {
		t.Fatal("fromBytesCached() reused a template after TemplateSet options changed")
	}
	if !tplWithTrim.Options.TrimBlocks {
		t.Fatal("cached template did not snapshot updated TrimBlocks option")
	}

	requireExecTemplateCacheStatsInternal(t, 0, 2, 0, 1)
}

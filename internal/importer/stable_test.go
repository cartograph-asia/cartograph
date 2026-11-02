package importer

import (
	"slices"
	"testing"
)

// A relation in Level.Exclude is never read, and the run counts it.
func TestLevelsExclude(t *testing.T) {
	imp := zzSettings()
	imp.District.Exclude = []int64{6753263}
	imp.Province.Exclude = []int64{2}
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "One"}, outer: []rect{{110, 8, 116, 12}}},
		{id: 2, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-02", "name", "Two"}, outer: []rect{{100, 8, 104, 12}}},
		{id: 6753263, tags: []string{"admin_level", "6", "name", "Nansha District"}, outer: []rect{{112, 9, 114, 11}}},
		{id: 7, tags: []string{"admin_level", "6", "name", "Seven"}, outer: []rect{{111, 9, 111.5, 9.5}}},
	}, nil)
	set := readTest(t, path, imp)
	if got := provinceIDs(set); !slices.Equal(got, []int64{1}) {
		t.Errorf("provinces %v", got)
	}
	if got := districtLevels(set); len(got) != 1 || got[7] != 6 {
		t.Errorf("districts %v", got)
	}
	if !slices.Equal(set.excluded, []int64{2, 6753263}) || set.relations != 2 {
		t.Errorf("excluded %v, relations read %d", set.excluded, set.relations)
	}
}

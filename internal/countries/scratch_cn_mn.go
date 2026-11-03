package countries

// TEMPORARY (scratch branch only, never pushed): rough settings for China and
// Mongolia so that WP N can measure the import's memory at their size.

func init() {
	cn := Imports["cn"]
	cn.Hold = ""
	cn.Province = Level{AdminLevels: []int{4}, ISOPrefix: "CN", Count: [2]int{20, 40}, Kind: "province"}
	cn.District = Level{AdminLevels: []int{6}, Kind: "county"}
	cn.English = []string{"name:en", "int_name", "name:zh-Latn-pinyin", "name"}
	cn.Local = []string{"name:zh", "name"}
	cn.LocalScripts = []string{"Han"}
	cn.LocalLang = "zh"
	cn.SlugSuffix = true
	cn.Neighbours = []string{"taiwan", "philippines"}
	Imports["cn"] = cn

	mn := Imports["mn"]
	mn.Hold = ""
	mn.Province = Level{AdminLevels: []int{4}, ISOPrefix: "MN", Count: [2]int{15, 30}, Kind: "province"}
	mn.District = Level{AdminLevels: []int{6}, Kind: "district"}
	mn.English = []string{"name:en", "int_name", "name"}
	mn.EnglishNot = []string{"Cyrillic"}
	mn.Local = []string{"name:mn", "name"}
	mn.LocalScripts = []string{"Cyrillic"}
	mn.LocalLang = "mn"
	mn.SlugSuffix = true
	Imports["mn"] = mn
}

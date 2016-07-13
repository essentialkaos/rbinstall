package index

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"sort"
	"strings"

	"pkg.re/essentialkaos/ek.v1/sortutil"
	"pkg.re/essentialkaos/ek.v1/system"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type VersionInfo struct {
	Name         string `json:"name"`
	File         string `json:"file"`
	Path         string `json:"path"`
	Size         uint64 `json:"size"`
	Hash         string `json:"hash"`
	RailsExpress bool   `json:"rx"`
}

type CategoryInfo struct {
	Versions []*VersionInfo `json:"versions"`
}

type CategoryData map[string]*CategoryInfo

type Data map[string]CategoryData

type Index struct {
	Data Data `json:"data"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

type versionInfoSlice []*VersionInfo

func (s versionInfoSlice) Len() int      { return len(s) }
func (s versionInfoSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s versionInfoSlice) Less(i, j int) bool {
	iv := strings.Replace(s[i].Name, "-", ".", -1)
	jv := strings.Replace(s[j].Name, "-", ".", -1)

	return sortutil.VersionCompare(iv, jv)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// NewIndex create new index struct
func NewIndex() *Index {
	return &Index{Data: make(Data)}
}

// NewCategoryInfo create new category info struct
func NewCategoryInfo() *CategoryInfo {
	return &CategoryInfo{make([]*VersionInfo, 0)}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Find info by name
func (i *Index) Find(name string) (*VersionInfo, string) {
	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		return nil, ""
	}

	for categoryName, category := range i.Data[systemInfo.Arch] {
		info := category.Find(name)

		if info != nil {
			return info, categoryName
		}
	}

	return nil, ""
}

// Find info by name
func (i *CategoryInfo) Find(name string) *VersionInfo {
	if len(i.Versions) == 0 {
		return nil
	}

	for _, info := range i.Versions {
		if info.Name == name {
			return info
		}
	}

	return nil
}

// Sort sorts versions data
func (i *Index) Sort() {
	for _, cData := range i.Data {
		for _, cInfo := range cData {
			sort.Sort(versionInfoSlice(cInfo.Versions))
		}
	}
}

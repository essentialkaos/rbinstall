package index

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
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
func (i *Index) Find(name string) *VersionInfo {
	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		return nil
	}

	for _, category := range i.Data[systemInfo.Arch] {
		info := category.Find(name)

		if info != nil {
			return info
		}
	}

	return nil
}

// Find info by name
func (ci *CategoryInfo) Find(name string) *VersionInfo {
	if len(ci.Versions) == 0 {
		return nil
	}

	for _, info := range ci.Versions {
		if info.Name == name {
			return info
		}
	}

	return nil
}

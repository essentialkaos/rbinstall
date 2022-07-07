package index

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2022 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/sortutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	CATEGORY_RUBY    = "ruby"
	CATEGORY_JRUBY   = "jruby"
	CATEGORY_TRUFFLE = "truffle"
	CATEGORY_OTHER   = "other"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Index struct {
	Meta    *Metadata           `json:"meta"`
	Data    map[string]DistData `json:"data"`
	Aliases map[string]string   `json:"aliases,omitempty"`
}

type Metadata struct {
	Created int64 `json:"created"` // Index creation timestamp
	Size    int64 `json:"size"`    // Total data size
	Items   int   `json:"items"`   // Total number of items in repo
}

type DistData map[string]ArchData

type ArchData map[string]CategoryData

type CategoryData []*VersionInfo

type VersionInfo struct {
	Variations []*VersionInfo `json:"variations,omitempty"` // Info about version variations (railsexpress version for example)
	Name       string         `json:"name"`                 // Version name
	File       string         `json:"file"`                 // Full filename (with extension)
	Path       string         `json:"path"`                 // Relative path to 7z file
	Hash       string         `json:"hash"`                 // SHA-256 hash
	Size       int64          `json:"size"`                 // Size in bytes
	Added      int64          `json:"added"`                // Timestamp with date when version was added to repo
	EOL        bool           `json:"eol"`                  // EOL marker
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

// NewIndex return pointer to new index struct
func NewIndex() *Index {
	return &Index{
		Meta: &Metadata{},
		Data: make(map[string]DistData),
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Add adds info about ruby to index
func (i *Index) Add(dist, arch, category string, info *VersionInfo) {
	if i == nil {
		return
	}

	if i.Data[dist] == nil {
		i.Data[dist] = make(DistData)
	}

	if i.Data[dist][arch] == nil {
		i.Data[dist][arch] = make(ArchData)
	}

	if i.Data[dist][arch][category] == nil {
		i.Data[dist][arch][category] = make(CategoryData, 0)
	}

	i.Data[dist][arch][category] = append(
		i.Data[dist][arch][category], info,
	)
}

// HasData returns true if index contains data for some dist + arch
func (i *Index) HasData(dist, arch string) bool {
	if i.Aliases[dist] != "" {
		dist = i.Aliases[dist]
	}

	if i.Data[dist] == nil {
		return false
	}

	if i.Data[dist][arch] == nil {
		return false
	}

	return true
}

// GetCategoryData returns data for given dist, arch and category
func (i *Index) GetCategoryData(dist, arch, category string, eol bool) CategoryData {
	if !i.HasData(dist, arch) {
		return nil
	}

	if i.Aliases[dist] != "" {
		dist = i.Aliases[dist]
	}

	if eol {
		return i.Data[dist][arch][category]
	}

	var result = CategoryData{}

	for _, v := range i.Data[dist][arch][category] {
		if v.EOL {
			continue
		}

		result = append(result, v)
	}

	return result
}

// Encode encodes index to JSON format
func (i *Index) Encode() ([]byte, error) {
	if i == nil {
		return nil, errors.New("Index is nil")
	}

	// Prepare index for encoding
	i.Sort()
	i.UpdateMetadata()

	data, err := json.MarshalIndent(i, "", "  ")

	if err != nil {
		return nil, err
	}

	return data, nil
}

// UpdateMetadata updates index metadata
func (i *Index) UpdateMetadata() {
	if i == nil {
		return
	}

	for _, dist := range i.Data {
		for _, arch := range dist {
			for _, category := range arch {
				for _, version := range category {

					i.Meta.Items++
					i.Meta.Size += version.Size

					if len(version.Variations) != 0 {
						for _, subver := range version.Variations {
							i.Meta.Items++
							i.Meta.Size += subver.Size
						}
					}
				}
			}
		}
	}

	i.Meta.Created = time.Now().Unix()
}

// Sort sorts versions data
func (i *Index) Sort() {
	if i == nil {
		return
	}

	for _, dist := range i.Data {
		for _, arch := range dist {
			for _, category := range arch {
				sort.Sort(versionInfoSlice(category))
			}
		}
	}
}

// Find tries to find info about version by name
func (i *Index) Find(dist, arch, name string) (*VersionInfo, string) {
	if i == nil {
		return nil, ""
	}

	if i.Aliases[dist] != "" {
		dist = i.Aliases[dist]
	}

	if i.Data[dist] == nil {
		return nil, ""
	}

	if i.Data[dist][arch] == nil {
		return nil, ""
	}

	for categoryName, category := range i.Data[dist][arch] {
		for i := len(category) - 1; i >= 0; i-- {
			version := category[i]

			if isSameName(version.Name, name) {
				return version, categoryName
			}

			if len(version.Variations) != 0 {
				for _, variation := range version.Variations {
					if isSameName(variation.Name, name) {
						return variation, categoryName
					}
				}
			}
		}
	}

	return nil, ""
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Total returns total number of rubies available for installation
func (d CategoryData) Total() int {
	if len(d) == 0 {
		return 0
	}

	var result int

	for _, v := range d {
		result += len(v.Variations) + 1
	}

	return result
}

// ////////////////////////////////////////////////////////////////////////////////// //

func isSameName(name1, name2 string) bool {
	if name1 == name2 {
		return true
	}

	if strings.Contains(name1, "-p") {
		nameSlice := strings.Split(name1, "-")

		switch len(nameSlice) {
		case 3:
			return nameSlice[0]+"-"+nameSlice[2] == name2
		case 2:
			return nameSlice[0] == name2
		}
	}

	return false
}

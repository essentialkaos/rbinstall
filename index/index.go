package index

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v7/sortutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Index struct {
	Meta *Metadata           `json:"meta"`
	Data map[string]DistData `json:"data"`
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
	Name  string `json:"name"`  // Version name
	File  string `json:"file"`  // Full filename (with extension)
	Path  string `json:"path"`  // Relative path to 7z file
	Size  int64  `json:"size"`  // Size in bytes
	Hash  string `json:"hash"`  // SHA-256 hash
	Added int64  `json:"added"` // Timestamp with date when version was added to repo

	// Variations contains info about version variations (railsexpress version
	// for example)
	Variations []*VersionInfo `json:"variations,omitempty"`
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

// Add used for adding info about some ruby version
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

// HasData return true if index contains data for
// some dist + arch
func (i *Index) HasData(dist, arch string) bool {
	if i.Data[dist] == nil {
		return false
	}

	if i.Data[dist][arch] == nil {
		return false
	}

	return true
}

// Encode encode index to JSON format
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

// UpdateMetadata update index metadata
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

// Find try to find info about version by name
func (i *Index) Find(dist, arch, name string) (*VersionInfo, string) {
	if i == nil {
		return nil, ""
	}

	if i.Data[dist] == nil {
		return nil, ""
	}

	if i.Data[dist][arch] == nil {
		return nil, ""
	}

	for categoryName, category := range i.Data[dist][arch] {
		for _, version := range category {
			if version.Name == name {
				return version, categoryName
			}

			if len(version.Variations) != 0 {
				for _, variation := range version.Variations {
					if variation.Name == name {
						return variation, categoryName
					}
				}
			}
		}
	}

	return nil, ""
}

// ////////////////////////////////////////////////////////////////////////////////// //

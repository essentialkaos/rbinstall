package index

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v3/sortutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Index struct {
	Meta *Metadata         `json:"meta"`
	Data map[string]OSData `json:"data"`
}

type Metadata struct {
	Created int64 `json:"created"` // Index creation timestamp
	Size    int64 `json:"size"`    // Total data size
	Items   int   `json:"items"`   // Total number of items in repo
}

type OSData map[string]ArchData

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
		Data: make(map[string]OSData),
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Add used for adding info about some ruby version
func (i *Index) Add(osName, archName, categoryName string, info *VersionInfo) {
	if i == nil {
		return
	}

	if i.Data[osName] == nil {
		i.Data[osName] = make(OSData)
	}

	if i.Data[osName][archName] == nil {
		i.Data[osName][archName] = make(ArchData)
	}

	if i.Data[osName][archName][categoryName] == nil {
		i.Data[osName][archName][categoryName] = make(CategoryData, 0)
	}

	i.Data[osName][archName][categoryName] = append(
		i.Data[osName][archName][categoryName], info,
	)
}

// HasData return true if index contains data for
// some os + arch
func (i *Index) HasData(osName, archName string) bool {
	if i.Data[osName] == nil {
		return false
	}

	if i.Data[osName][archName] == nil {
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

	for _, os := range i.Data {
		for _, arch := range os {
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

	for _, os := range i.Data {
		for _, arch := range os {
			for _, category := range arch {
				sort.Sort(versionInfoSlice(category))
			}
		}
	}
}

// Find try to find info about version by name
func (i *Index) Find(os, arch, name string) (*VersionInfo, string) {
	if i == nil {
		return nil, ""
	}

	if i.Data[os] == nil {
		return nil, ""
	}

	if i.Data[os][arch] == nil {
		return nil, ""
	}

	for categoryName, category := range i.Data[os][arch] {
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

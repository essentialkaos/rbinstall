package gen

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"os"
	"sort"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/crypto"
	"pkg.re/essentialkaos/ek.v1/fmtc"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/jsonutil"
	"pkg.re/essentialkaos/ek.v1/sliceutil"
	"pkg.re/essentialkaos/ek.v1/timeutil"
	"pkg.re/essentialkaos/ek.v1/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall Gen"
	VER  = "0.4.3"
	DESC = "Utility for generating RBInstall index"
)

const (
	ARG_OUTPUT   = "o:output"
	ARG_NO_COLOR = "nc:no-color"
	ARG_HELP     = "h:help"
	ARG_VER      = "v:version"
)

const (
	CATEGORY_RUBY     = "ruby"
	CATEGORY_JRUBY    = "jruby"
	CATEGORY_REE      = "ree"
	CATEGORY_RUBINIUS = "rubinius"
	CATEGORY_OTHER    = "other"
)

// ////////////////////////////////////////////////////////////////////////////////// //

var argList = arg.Map{
	ARG_OUTPUT:   &arg.V{},
	ARG_NO_COLOR: &arg.V{Type: arg.BOOL},
	ARG_HELP:     &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:      &arg.V{Type: arg.BOOL, Alias: "ver"},
}

var archList []string = []string{"i386", "x86_64"}

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	args, errs := arg.Parse(argList)

	if len(errs) != 0 {
		for _, err := range errs {
			fmtc.Printf("{r}%s{!}\n", err.Error())
		}

		os.Exit(1)
	}

	if arg.GetB(ARG_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if arg.GetB(ARG_VER) {
		showAbout()
		return
	}

	if arg.GetB(ARG_HELP) || len(args) == 0 {
		showUsage()
		return
	}

	path := args[0]

	checkDir(path)
	buildIndex(path)
}

// checkDir do some checks for provided dir
func checkDir(path string) {
	if !fsutil.IsDir(path) {
		fmtc.Printf("{r}Target %s is not a directory{!}\n", path)
		os.Exit(1)
	}

	if !fsutil.IsExist(path) {
		fmtc.Printf("{r}Directory %s is not exist{!}\n", path)
		os.Exit(1)
	}

	if !fsutil.IsReadable(path) {
		fmtc.Printf("{r}Directory %s is not readable{!}\n", path)
		os.Exit(1)
	}

	if !fsutil.IsExecutable(path) {
		fmtc.Printf("{r}Directory %s is not exectable{!}\n", path)
		os.Exit(1)
	}

	if arg.GetS(ARG_OUTPUT) == "" && !fsutil.IsWritable(path) {
		fmtc.Printf("{r}Directory %s is not writable{!}\n", path)
		os.Exit(1)
	}
}

// buildIndex create index
func buildIndex(path string) {
	var err error

	dirList := fsutil.List(path, true)

	if len(dirList) == 0 {
		fmtc.Println("\n{y}Can't find arch directories in specified directory{!}\n")
		os.Exit(0)
	}

	outputFile := arg.GetS(ARG_OUTPUT)

	if outputFile == "" {
		outputFile = path + "/index.json"
	}

	var (
		newIndex *index.Index
		oldIndex *index.Index
	)

	newIndex = index.NewIndex()

	// Reuse index if possible
	if fsutil.IsExist(outputFile) {
		oldIndex = index.NewIndex()

		err = jsonutil.DecodeFile(outputFile, oldIndex)

		if err != nil {
			fmtc.Printf("\n{y}Can't reuse existing index: %s{!}\n\n", err.Error())
			oldIndex = nil
		}
	}

	start := time.Now()

	for _, dir := range dirList {
		if !fsutil.IsDir(path + "/" + dir) {
			continue
		}

		arch := dir

		if !sliceutil.Contains(archList, arch) {
			fmtc.Printf("\n{y}Unknown arch %s. Skipping...\n\n", arch)
			continue
		}

		if newIndex.Data[arch] == nil {
			newIndex.Data[arch] = make(index.CategoryData)
		}

		fileList := fsutil.List(path+"/"+arch, true)

		sort.Strings(fileList)

		if len(fileList) == 0 {
			fmtc.Printf("\n{y}Can't find files in %s directory. Skipping...{!}\n\n", path+"/"+arch)
			continue
		}

		for _, file := range fileList {
			if !fsutil.IsRegular(path + "/" + arch + "/" + file) {
				continue
			}

			category := CATEGORY_OTHER

			switch {
			case file[0:2] == "1.", file[0:2] == "2.", file[0:3] == "dev":
				category = CATEGORY_RUBY
			case file[0:5] == "jruby":
				category = CATEGORY_JRUBY
			case file[0:3] == "ree":
				category = CATEGORY_REE
			case file[0:5] == "rubin":
				category = CATEGORY_RUBINIUS
			}

			if newIndex.Data[arch][category] == nil {
				newIndex.Data[arch][category] = index.NewCategoryInfo()
			}

			cleanName := strings.Replace(file, ".7z", "", -1)
			fileSize := uint64(fsutil.GetSize(path + "/" + arch + "/" + file))

			info := findInfo(oldIndex.Data[arch][category].Versions, cleanName)

			if info == nil || info.Size != fileSize {
				info = &index.VersionInfo{
					Name:         cleanName,
					File:         file,
					Path:         "/" + arch + "/" + file,
					Size:         fileSize,
					Hash:         crypto.FileHash(path + "/" + arch + "/" + file),
					RailsExpress: fsutil.IsExist(path + "/" + arch + "/" + cleanName + "-railsexpress.7z"),
				}

				fmtc.Printf("{g}+ %-24s{!} → {c}%s/%s{!}\n", info.Name, arch, category)
			} else {
				fmtc.Printf("{s}+ %-24s → %s/%s{!}\n", info.Name, arch, category)
			}

			newIndex.Data[arch][category].Versions = append(newIndex.Data[arch][category].Versions, info)
		}
	}

	fmtc.NewLine()

	if fsutil.IsExist(outputFile) {
		os.RemoveAll(outputFile)
	}

	err = jsonutil.EncodeToFile(outputFile, newIndex)

	if err != nil {
		fmtc.Printf("{r}Can't save index as file %s: %s{!}\n", outputFile, err.Error())
	} else {
		fmtc.Printf("{g}Index created and stored as file %s. Processing took %s{!}\n", outputFile, timeutil.PrettyDuration(time.Since(start)))
	}

	fmtc.NewLine()
}

// findInfo search version info struct in given slice
func findInfo(infoList []*index.VersionInfo, version string) *index.VersionInfo {
	for _, info := range infoList {
		if info.Name == version {
			return info
		}
	}

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

func showUsage() {
	info := usage.NewInfo("", "dir")

	info.AddOption(ARG_OUTPUT, "Custom index output", "file")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VER, "Show version")

	info.AddExample("/dir/with/rubies")
	info.AddExample("-o index.json /dir/with/rubies")

	info.Render()
}

func showAbout() {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2006,
		Owner:   "ESSENTIAL KAOS",
		License: "Essential Kaos Open Source License <https://essentialkaos.com/ekol?en>",
	}

	about.Render()
}

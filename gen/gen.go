package gen

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/crypto"
	"pkg.re/essentialkaos/ek.v1/fmtc"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/jsonutil"
	"pkg.re/essentialkaos/ek.v1/path"
	"pkg.re/essentialkaos/ek.v1/sliceutil"
	"pkg.re/essentialkaos/ek.v1/timeutil"
	"pkg.re/essentialkaos/ek.v1/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall Gen"
	VER  = "0.4.5"
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

var archList = []string{"i386", "x86_64"}

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	runtime.GOMAXPROCS(1)

	args, errs := arg.Parse(argList)

	if len(errs) != 0 {
		for _, err := range errs {
			printError(err.Error())
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

	dataDir := args[0]

	checkDir(dataDir)
	buildIndex(dataDir)
}

// checkDir do some checks for provided dir
func checkDir(dataDir string) {
	if !fsutil.IsDir(dataDir) {
		printError("Target %s is not a directory", dataDir)
		os.Exit(1)
	}

	if !fsutil.IsExist(dataDir) {
		printError("Directory %s is not exist", dataDir)
		os.Exit(1)
	}

	if !fsutil.IsReadable(dataDir) {
		printError("Directory %s is not readable", dataDir)
		os.Exit(1)
	}

	if !fsutil.IsExecutable(dataDir) {
		printError("Directory %s is not exectable", dataDir)
		os.Exit(1)
	}

	if arg.GetS(ARG_OUTPUT) == "" && !fsutil.IsWritable(dataDir) {
		printError("Directory %s is not writable", dataDir)
		os.Exit(1)
	}
}

// buildIndex create index
func buildIndex(dataDir string) {
	var err error

	dirList := fsutil.List(dataDir, true)

	if len(dirList) == 0 {
		printWarn("\nCan't find arch directories in specified directory")
		os.Exit(0)
	}

	outputFile := arg.GetS(ARG_OUTPUT)

	if outputFile == "" {
		outputFile = path.Join(dataDir, "index.json")
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
			printWarn("\nCan't reuse existing index: %v\n", err)
			oldIndex = nil
		}
	}

	start := time.Now()

	for _, dir := range dirList {
		if !fsutil.IsDir(path.Join(dataDir, dir)) {
			continue
		}

		arch := dir

		if !sliceutil.Contains(archList, arch) {
			printWarn("\nUnknown arch %s. Skipping...\n", arch)
			continue
		}

		if newIndex.Data[arch] == nil {
			newIndex.Data[arch] = make(index.CategoryData)
		}

		fileList := fsutil.List(path.Join(dataDir, arch), true)

		sort.Strings(fileList)

		if len(fileList) == 0 {
			printWarn("\nCan't find files in %s directory. Skipping...\n", dataDir+"/"+arch)
			continue
		}

		for _, file := range fileList {
			filePath := path.Join(dataDir, arch, file)

			if !fsutil.IsRegular(filePath) {
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
			fileSize := uint64(fsutil.GetSize(filePath))

			info := findInfo(oldIndex.Data[arch][category].Versions, cleanName)

			if info == nil || info.Size != fileSize {
				info = &index.VersionInfo{
					Name:         cleanName,
					File:         file,
					Path:         "/" + arch + "/" + file,
					Size:         fileSize,
					Hash:         crypto.FileHash(filePath),
					RailsExpress: strings.Contains(filePath, "-railsexpress"),
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

	newIndex.Sort()

	err = jsonutil.EncodeToFile(outputFile, newIndex)

	if err != nil {
		printWarn("Can't save index as file %s: %v", outputFile, err)
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

// printError prints error message to console
func printError(f string, a ...interface{}) {
	fmtc.Printf("{r}"+f+"{!}\n", a...)
}

// printError prints warning message to console
func printWarn(f string, a ...interface{}) {
	fmtc.Printf("{y}"+f+"{!}\n", a...)
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

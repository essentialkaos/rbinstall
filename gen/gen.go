package gen

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/fsutil"
	"pkg.re/essentialkaos/ek.v9/hash"
	"pkg.re/essentialkaos/ek.v9/jsonutil"
	"pkg.re/essentialkaos/ek.v9/options"
	"pkg.re/essentialkaos/ek.v9/path"
	"pkg.re/essentialkaos/ek.v9/sortutil"
	"pkg.re/essentialkaos/ek.v9/timeutil"
	"pkg.re/essentialkaos/ek.v9/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall Gen"
	VER  = "0.8.0"
	DESC = "Utility for generating RBInstall index"
)

const (
	OPT_OUTPUT   = "o:output"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"
)

const (
	CATEGORY_RUBY     = "ruby"
	CATEGORY_JRUBY    = "jruby"
	CATEGORY_REE      = "ree"
	CATEGORY_RUBINIUS = "rubinius"
	CATEGORY_OTHER    = "other"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type FileInfo struct {
	OS       string
	Arch     string
	Category string
	File     string
}

type fileInfoSlice []FileInfo

func (s fileInfoSlice) Len() int      { return len(s) }
func (s fileInfoSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s fileInfoSlice) Less(i, j int) bool {
	iv := strings.Replace(s[i].File, "-", ".", -1)
	jv := strings.Replace(s[j].File, "-", ".", -1)

	return sortutil.VersionCompare(iv, jv)
}

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_OUTPUT:   {},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:      {Type: options.BOOL, Alias: "ver"},
}

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	runtime.GOMAXPROCS(1)

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		for _, err := range errs {
			printError(err.Error())
		}

		os.Exit(1)
	}

	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if options.GetB(OPT_VER) {
		showAbout()
		return
	}

	if options.GetB(OPT_HELP) || len(args) == 0 {
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
		printError("Directory %s does not exist", dataDir)
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

	if options.GetS(OPT_OUTPUT) == "" && !fsutil.IsWritable(dataDir) {
		printError("Directory %s is not writable", dataDir)
		os.Exit(1)
	}
}

// buildIndex create index
func buildIndex(dataDir string) {
	fmtc.NewLine()

	fileList := fsutil.ListAllFiles(
		dataDir, true,
		fsutil.ListingFilter{
			Perms:         "FR",
			MatchPatterns: []string{"*.7z"},
		})

	if len(fileList) == 0 {
		printWarn("Can't find any data in given directory\n")
		os.Exit(1)
	}

	outputFile := getOutputFile(dataDir)

	var (
		newIndex = index.NewIndex()
		oldIndex = getExistentIndex(outputFile)
	)

	start := time.Now()

	for _, fileInfo := range processFiles(fileList) {
		alreadyExist := false

		filePath := path.Join(dataDir, fileInfo.OS, fileInfo.Arch, fileInfo.File)
		fileName := strings.Replace(fileInfo.File, ".7z", "", -1)
		fileSize := fsutil.GetSize(filePath)
		fileAdded, _ := fsutil.GetCTime(filePath)

		versionInfo := &index.VersionInfo{
			Name:  fileName,
			File:  fileName + ".7z",
			Path:  path.Join(fileInfo.OS, fileInfo.Arch),
			Size:  fileSize,
			Added: fileAdded.Unix(),
		}

		oldVersionInfo, _ := oldIndex.Find(fileInfo.OS, fileInfo.Arch, fileName)

		// If 7z file have same creation date and size, we use hash from old index
		if oldVersionInfo != nil {
			if oldVersionInfo.Added == fileAdded.Unix() && oldVersionInfo.Size == fileSize {
				versionInfo.Hash = oldVersionInfo.Hash
				alreadyExist = true
			}
		}

		// Calculate hash if is not set
		if versionInfo.Hash == "" {
			versionInfo.Hash = hash.FileHash(filePath)
		}

		if strings.HasSuffix(fileName, "-railsexpress") {
			baseVersionName := strings.Replace(fileName, "-railsexpress", "", -1)
			baseVersionInfo, _ := newIndex.Find(fileInfo.OS, fileInfo.Arch, baseVersionName)

			if baseVersionInfo == nil {
				printWarn("Can't find base version info for %s", fileName)
				continue
			}

			baseVersionInfo.Variations = append(baseVersionInfo.Variations, versionInfo)
		} else {
			newIndex.Add(fileInfo.OS, fileInfo.Arch, fileInfo.Category, versionInfo)
		}

		if alreadyExist {
			fmtc.Printf(
				"{s-}  %-24s → %s/%s %s{!}\n",
				fileName, fileInfo.OS,
				fileInfo.Arch, fileInfo.Category,
			)
		} else {
			fmtc.Printf(
				"{g}+ %-24s{!} → {c}%s/%s %s{!}\n",
				fileName, fileInfo.OS,
				fileInfo.Arch, fileInfo.Category,
			)
		}
	}

	fmtc.NewLine()

	saveIndex(outputFile, newIndex)

	fmtc.Printf(
		"{g}Index created and stored as file {g*}%s{g}. Processing took %s{!}\n\n",
		outputFile, timeutil.PrettyDuration(time.Since(start)),
	)
}

// processFiles parse file list to FileInfo slice
func processFiles(files []string) []FileInfo {
	var result []FileInfo

	for _, file := range files {
		fileInfoSlice := strings.Split(file, "/")

		if len(fileInfoSlice) != 3 {
			continue
		}

		result = append(result, FileInfo{
			OS:       fileInfoSlice[0],
			Arch:     fileInfoSlice[1],
			Category: guessCategory(fileInfoSlice[2]),
			File:     fileInfoSlice[2],
		})
	}

	sort.Sort(fileInfoSlice(result))

	return result
}

// saveIndex save index data as JSON to file
func saveIndex(outputFile string, i *index.Index) {
	indexData, err := i.Encode()

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if fsutil.IsExist(outputFile) {
		os.RemoveAll(outputFile)
	}

	err = ioutil.WriteFile(outputFile, indexData, 0644)

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}
}

// guessCategory try to guess category by file name
func guessCategory(name string) string {
	switch {
	case name[0:2] == "1.", name[0:2] == "2.":
		return CATEGORY_RUBY
	case name[0:5] == "jruby":
		return CATEGORY_JRUBY
	case name[0:3] == "ree":
		return CATEGORY_REE
	case name[0:5] == "rubin":
		return CATEGORY_RUBINIUS
	}

	return CATEGORY_OTHER
}

// getOutputFile return path to output file
func getOutputFile(dataDir string) string {
	outputFile := options.GetS(OPT_OUTPUT)

	if outputFile != "" {
		return outputFile
	}

	return path.Join(dataDir, "index.json")
}

// getExistentIndex read and decode index
func getExistentIndex(file string) *index.Index {
	if !fsutil.IsExist(file) {
		fmtc.Println("{s-}An earlier version of index is not found{!}\n")
		return nil
	}

	i := index.NewIndex()

	err := jsonutil.DecodeFile(file, i)

	if err != nil {
		printWarn("Can't reuse existing index: %v\n", err)
		return nil
	}

	return i
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

	info.AddOption(OPT_OUTPUT, "Custom index output", "file")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

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

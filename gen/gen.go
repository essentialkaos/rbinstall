package gen

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2019 ESSENTIAL KAOS                         //
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

	"pkg.re/essentialkaos/ek.v11/fmtc"
	"pkg.re/essentialkaos/ek.v11/fsutil"
	"pkg.re/essentialkaos/ek.v11/hash"
	"pkg.re/essentialkaos/ek.v11/jsonutil"
	"pkg.re/essentialkaos/ek.v11/options"
	"pkg.re/essentialkaos/ek.v11/path"
	"pkg.re/essentialkaos/ek.v11/sortutil"
	"pkg.re/essentialkaos/ek.v11/strutil"
	"pkg.re/essentialkaos/ek.v11/timeutil"
	"pkg.re/essentialkaos/ek.v11/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "RBInstall Gen"
	VER  = "0.11.0"
	DESC = "Utility for generating RBInstall index"
)

// Options
const (
	OPT_OUTPUT   = "o:output"
	OPT_EOL      = "e:eol"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"
)

// Categories
const (
	CATEGORY_RUBY     = "ruby"
	CATEGORY_JRUBY    = "jruby"
	CATEGORY_REE      = "ree"
	CATEGORY_RUBINIUS = "rubinius"
	CATEGORY_OTHER    = "other"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// FileInfo contains info about file
type FileInfo struct {
	OS       string
	Arch     string
	Category string
	File     string
}

// ////////////////////////////////////////////////////////////////////////////////// //

type fileInfoSlice []FileInfo

func (s fileInfoSlice) Len() int      { return len(s) }
func (s fileInfoSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s fileInfoSlice) Less(i, j int) bool {
	iv := strings.Replace(s[i].File, "-", ".", -1)
	jv := strings.Replace(s[j].File, "-", ".", -1)

	return sortutil.VersionCompare(iv, jv)
}

// ////////////////////////////////////////////////////////////////////////////////// //

var eolInfo map[string]bool

var optMap = options.Map{
	OPT_OUTPUT:   {},
	OPT_EOL:      {},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:      {Type: options.BOOL, Alias: "ver"},
}

var variations = []string{"railsexpress", "jemalloc"}

// ////////////////////////////////////////////////////////////////////////////////// //

// Init is main func
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

	loadEOLInfo()
	checkDir(dataDir)
	buildIndex(dataDir)
}

// loadEOLInfo load EOL info from file
func loadEOLInfo() {
	eolInfo = make(map[string]bool)

	if !options.Has(OPT_EOL) {
		return
	}

	err := jsonutil.Read(options.GetS(OPT_EOL), &eolInfo)

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// checkDir do some checks for provided dir
func checkDir(dataDir string) {
	if !fsutil.IsDir(dataDir) {
		printErrorAndExit("Target %s is not a directory", dataDir)
	}

	if !fsutil.IsExist(dataDir) {
		printErrorAndExit("Directory %s does not exist", dataDir)
	}

	if !fsutil.IsReadable(dataDir) {
		printErrorAndExit("Directory %s is not readable", dataDir)
	}

	if !fsutil.IsExecutable(dataDir) {
		printErrorAndExit("Directory %s is not exectable", dataDir)
	}

	if options.GetS(OPT_OUTPUT) == "" && !fsutil.IsWritable(dataDir) {
		printErrorAndExit("Directory %s is not writable", dataDir)
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
		printErrorAndExit("Can't find any data in given directory\n")
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
			EOL:   isEOLVersion(fileName),
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

		if isBaseRubyVariation(fileName) {
			baseVersionName := getVariationBaseName(fileName)
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
		"{g}Index created and stored as file {*}%s{!*}. Processing took %s{!}\n\n",
		outputFile, timeutil.PrettyDuration(time.Since(start)),
	)
}

// isEOLVersion return true if it EOL version
func isEOLVersion(name string) bool {
	if len(eolInfo) == 0 {
		return false
	}

	for version := range eolInfo {
		if strings.HasPrefix(name, version) {
			return true
		}
	}

	return false
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
		printErrorAndExit(err.Error())
	}

	if fsutil.IsExist(outputFile) {
		os.RemoveAll(outputFile + ".bak")
		fsutil.MoveFile(outputFile, outputFile+".bak", 0600)
	}

	err = ioutil.WriteFile(outputFile, indexData, 0644)

	if err != nil {
		printErrorAndExit(err.Error())
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

	err := jsonutil.Read(file, i)

	if err != nil {
		printWarn("Can't reuse existing index: %v\n", err)
		return nil
	}

	return i
}

// isBaseRubyVariation returns true if given name is name of base ruby variation
func isBaseRubyVariation(name string) bool {
	for _, v := range variations {
		if strings.HasSuffix(name, "-"+v) {
			return true
		}
	}

	return false
}

// getVariationBaseName returns base ruby name
func getVariationBaseName(name string) string {
	for _, v := range variations {
		name = strutil.Exclude(name, "-"+v)
	}

	return name
}

// printError prints error message to console
func printError(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
}

// printError prints warning message to console
func printWarn(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{y}"+f+"{!}\n", a...)
}

// printErrorAndExit print error message and exit with non-zero exit code
func printErrorAndExit(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

func showUsage() {
	info := usage.NewInfo("", "dir")

	info.AddOption(OPT_OUTPUT, "Custom index output", "file")
	info.AddOption(OPT_EOL, "File with EOL info", "file")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"/dir/with/rubies",
		"Generate index for directory /dir/with/rubies",
	)

	info.AddExample(
		"-o all.json /dir/with/rubies",
		"Generate index for directory /dir/with/rubies and save all all.json",
	)

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

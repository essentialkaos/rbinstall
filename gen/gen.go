package gen

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/hash"
	"github.com/essentialkaos/ek/v12/jsonutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/sortutil"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/support"
	"github.com/essentialkaos/ek/v12/support/deps"
	"github.com/essentialkaos/ek/v12/terminal/tty"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "RBInstall Gen"
	VER  = "3.2.4"
	DESC = "Utility for generating RBInstall index"
)

// Options
const (
	OPT_OUTPUT   = "o:output"
	OPT_EOL      = "e:eol"
	OPT_ALIAS    = "a:alias"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// INDEX_NAME is name of index file
const INDEX_NAME = "index3.json"

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
	return sortutil.VersionCompare(fmtVersionName(s[i].File), fmtVersionName(s[j].File))
}

// ////////////////////////////////////////////////////////////////////////////////// //

var eolInfo map[string]bool
var aliasInfo map[string]string

var optMap = options.Map{
	OPT_OUTPUT:   {Value: INDEX_NAME},
	OPT_EOL:      {Value: "eol.json"},
	OPT_ALIAS:    {Value: "alias.json"},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL},
	OPT_VER:      {Type: options.MIXED},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

var variations = []string{"railsexpress", "jemalloc"}

var colorTagApp, colorTagVer string

// ////////////////////////////////////////////////////////////////////////////////// //

func Run(gitRev string, gomod []byte) {
	runtime.GOMAXPROCS(1)

	preConfigureUI()

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		printError(errs[0].Error())
		os.Exit(1)
	}

	configureUI()

	switch {
	case options.Has(OPT_COMPLETION):
		os.Exit(printCompletion())
	case options.Has(OPT_GENERATE_MAN):
		printMan()
		os.Exit(0)
	case options.GetB(OPT_VER):
		genAbout(gitRev).Print(options.GetS(OPT_VER))
		os.Exit(0)
	case options.GetB(OPT_VERB_VER):
		support.Collect(APP, VER).WithRevision(gitRev).
			WithDeps(deps.Extract(gomod)).Print()
		os.Exit(0)
	case options.GetB(OPT_HELP) || len(args) == 0:
		genUsage().Print()
		os.Exit(0)
	}

	dataDir := args.Get(0).Clean().String()

	loadEOLInfo()
	loadAliasInfo()
	checkDir(dataDir)
	buildIndex(dataDir)
}

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !fmtc.IsColorsSupported() {
		fmtc.DisableColors = true
	}

	if !tty.IsTTY() {
		fmtc.DisableColors = true
	}
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{*}{#CC1E2C}", "{#CC1E2C}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#160}", "{#160}"
	default:
		colorTagApp, colorTagVer = "{*}{r}", "{r}"
	}
}

// loadEOLInfo loads EOL info from file
func loadEOLInfo() {
	eolInfo = make(map[string]bool)

	if !fsutil.CheckPerms("FRS", options.GetS(OPT_EOL)) {
		if !options.Has(OPT_EOL) {
			return
		}
	}

	err := jsonutil.Read(options.GetS(OPT_EOL), &eolInfo)

	if err != nil {
		printErrorAndExit("Can't read EOL data: %v", err)
	}
}

// loadAliasInfo loads aliases info
func loadAliasInfo() {
	aliasInfo = make(map[string]string)

	if !fsutil.CheckPerms("FRS", options.GetS(OPT_ALIAS)) {
		if !options.Has(OPT_ALIAS) {
			return
		}
	}

	err := jsonutil.Read(options.GetS(OPT_ALIAS), &aliasInfo)

	if err != nil {
		printErrorAndExit("Can't read alias data: %v", err)
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
		printErrorAndExit("Directory %s is not executable", dataDir)
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
			Perms:         "FRS",
			MatchPatterns: []string{"*.tzst"},
		})

	if len(fileList) == 0 {
		printErrorAndExit("Can't find any data in given directory\n")
	}

	outputFile := options.GetS(OPT_OUTPUT)
	newIndex := index.NewIndex()
	oldIndex := getExistentIndex(outputFile)

	start := time.Now()

	for _, fileInfo := range processFiles(fileList) {
		alreadyExist := false

		filePath := path.Join(dataDir, fileInfo.OS, fileInfo.Arch, fileInfo.File)
		fileName := strutil.Exclude(fileInfo.File, ".tzst")
		fileSize := fsutil.GetSize(filePath)
		fileAdded, _ := fsutil.GetCTime(filePath)

		versionInfo := &index.VersionInfo{
			Name:  fileName,
			File:  fileName + ".tzst",
			Path:  path.Join(fileInfo.OS, fileInfo.Arch),
			Size:  fileSize,
			Added: fileAdded.Unix(),
			EOL:   isEOLVersion(fileName),
		}

		oldVersionInfo, _ := oldIndex.Find(fileInfo.OS, fileInfo.Arch, fileName)

		// If file have same creation date and size, we use hash from old index
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
				"{g}+ %-24s{!} → {c}%s/%s {c*}%s{!}\n",
				fileName, fileInfo.OS,
				fileInfo.Arch, fileInfo.Category,
			)
		}
	}

	if len(aliasInfo) != 0 {
		newIndex.Aliases = aliasInfo
	}

	printIndexStats(newIndex)
	printExtraInfo()

	fmtutil.Separator(false)

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
			File:     fileInfoSlice[2],
			Category: guessCategory(fileInfoSlice[2]),
		})
	}

	sort.Sort(fileInfoSlice(result))

	return result
}

// printIndexStats prints index statistics
func printIndexStats(i *index.Index) {
	fmtutil.Separator(false, "STATISTICS")

	i.UpdateMeta()

	for _, distName := range i.Data.Keys() {
		size := int64(0)
		items := 0

		for archName, arch := range i.Data[distName] {
			for _, category := range arch {
				for _, version := range category {
					size += version.Size
					items++

					if len(version.Variations) != 0 {
						for _, variation := range version.Variations {
							items++
							size += variation.Size
						}
					}
				}
			}

			fmtc.Printf(
				"  {c*}%s{!}{c}/%s:{!} %3s {s-}|{!} %s\n", distName, archName,
				fmtutil.PrettyNum(items), fmtutil.PrettySize(size, " "),
			)
		}
	}

	fmtc.NewLine()
	fmtc.Printf(
		"  {*}Total:{!} %s {s-}|{!} %s\n",
		fmtutil.PrettyNum(i.Meta.Items),
		fmtutil.PrettySize(i.Meta.Size, " "),
	)
}

// printExtraInfo prints info about used alias/eol data
func printExtraInfo() {
	fmtutil.Separator(false, "EXTRA")

	eolModTime, _ := fsutil.GetMTime(options.GetS(OPT_EOL))
	aliasModTime, _ := fsutil.GetMTime(options.GetS(OPT_ALIAS))

	fmtc.If(len(eolInfo) == 0).Println("  {*}EOL:  {!} {s}—{!}")
	fmtc.If(len(eolInfo) != 0).Printf(
		"  {*}EOL:  {!} %s {s-}(%s){!}\n",
		options.GetS(OPT_EOL),
		timeutil.Format(eolModTime, "%Y/%m/%d %H:%M"),
	)

	fmtc.If(len(aliasInfo) == 0).Println("  {*}Alias:{!} {s}—{!}")
	fmtc.If(len(aliasInfo) != 0).Printf(
		"  {*}Alias:{!} %s {s-}(%s){!}\n",
		options.GetS(OPT_ALIAS),
		timeutil.Format(aliasModTime, "%Y/%m/%d %H:%M"),
	)
}

// saveIndex saves index data as JSON to file
func saveIndex(outputFile string, i *index.Index) {
	indexData, err := i.Encode()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if fsutil.IsExist(outputFile) {
		os.RemoveAll(outputFile + ".bkp")
		fsutil.MoveFile(outputFile, outputFile+".bkp", 0600)
	}

	err = os.WriteFile(outputFile, indexData, 0644)

	if err != nil {
		printErrorAndExit("Can't save index: %v", err)
	}

	os.Chmod(outputFile, 0644)
}

// guessCategory try to guess category by file name
func guessCategory(name string) string {
	switch {
	case strings.HasPrefix(name, "1."),
		strings.HasPrefix(name, "2."),
		strings.HasPrefix(name, "3."):
		return index.CATEGORY_RUBY
	case strings.HasPrefix(name, "jruby"):
		return index.CATEGORY_JRUBY
	case strings.HasPrefix(name, "truffle"):
		return index.CATEGORY_TRUFFLE
	}

	return index.CATEGORY_OTHER
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

// fmtVersionName formats version file name for comparison
func fmtVersionName(v string) string {
	v = strings.ReplaceAll(v, ".tzst", "")
	v = strings.ReplaceAll(v, "-p", ".")
	v = strings.ReplaceAll(v, "-railsexpress", ".1")
	v = strings.ReplaceAll(v, "-jemalloc", ".2")
	return v
}

// printError prints error message to console
func printError(f string, a ...any) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
}

// printError prints warning message to console
func printWarn(f string, a ...any) {
	fmtc.Fprintf(os.Stderr, "{y}"+f+"{!}\n", a...)
}

// printErrorAndExit print error message and exit with non-zero exit code
func printErrorAndExit(f string, a ...any) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printCompletion prints completion for given shell
func printCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Print(bash.Generate(info, "rbinstall-clone"))
	case "fish":
		fmt.Print(fish.Generate(info, "rbinstall-clone"))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, "rbinstall-clone"))
	default:
		return 1
	}

	return 0
}

// printMan prints man page
func printMan() {
	fmt.Println(
		man.Generate(
			genUsage(),
			genAbout(""),
		),
	)
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo("", "dir")

	info.AppNameColorTag = colorTagApp

	info.AddOption(OPT_OUTPUT, "Custom index output {s-}(default: index.json){!}", "file")
	info.AddOption(OPT_EOL, "File with EOL information {s-}(default: eol.json){!}", "file")
	info.AddOption(OPT_ALIAS, "File with aliases information {s-}(default: alias.json){!}", "file")
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

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2006,

		AppNameColorTag: colorTagApp,
		VersionColorTag: colorTagVer,
		DescSeparator:   "{s}—{!}",

		Owner:   "ESSENTIAL KAOS",
		License: "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
	}

	return about
}

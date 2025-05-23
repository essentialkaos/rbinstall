package clone

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/httputil"
	"github.com/essentialkaos/ek/v13/jsonutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/pluralize"
	"github.com/essentialkaos/ek/v13/progress"
	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/support"
	"github.com/essentialkaos/ek/v13/support/deps"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"
	"github.com/essentialkaos/ek/v13/terminal/tty"
	"github.com/essentialkaos/ek/v13/timeutil"
	"github.com/essentialkaos/ek/v13/usage"
	"github.com/essentialkaos/ek/v13/usage/completion/bash"
	"github.com/essentialkaos/ek/v13/usage/completion/fish"
	"github.com/essentialkaos/ek/v13/usage/completion/zsh"
	"github.com/essentialkaos/ek/v13/usage/man"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "RBInstall Clone"
	VER  = "3.2.1"
	DESC = "Utility for cloning RBInstall repository"
)

// Options
const (
	OPT_YES      = "y:yes"
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

// FileInfo contains info about file with Ruby data
type FileInfo struct {
	File string
	URL  string
	OS   string
	Arch string
	Size int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_YES:      {Type: options.BOOL},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL},
	OPT_VER:      {Type: options.MIXED},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

var colorTagApp, colorTagVer string

// ////////////////////////////////////////////////////////////////////////////////// //

func Run(gitRev string, gomod []byte) {
	runtime.GOMAXPROCS(1)

	preConfigureUI()

	args, errs := options.Parse(optMap)

	if !errs.IsEmpty() {
		terminal.Error("Options parsing errors:")
		terminal.Error(errs.Error(" - "))
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
	case options.GetB(OPT_HELP) || len(args) != 2:
		genUsage().Print()
		os.Exit(0)
	}

	url := args.Get(0).String()
	dir := args.Get(1).String()

	fmtc.NewLine()

	checkArguments(url, dir)
	cloneRepository(url, dir)

	fmtc.NewLine()
}

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() {
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

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	input.Prompt = "› "
	input.TitleColorTag = "{s}"
	input.NewLine = true

	req.SetUserAgent("RBInstall-Clone", VER)
}

// checkArguments checks command line arguments
func checkArguments(url, dir string) {
	if !httputil.IsURL(url) {
		printErrorAndExit("Url %s doesn't look like valid url", url)
	}

	if !fsutil.IsExist(dir) {
		printErrorAndExit("Directory %s does not exist", dir)
	}

	if !fsutil.IsDir(dir) {
		printErrorAndExit("Target %s is not a directory", dir)
	}

	if !fsutil.IsReadable(dir) {
		printErrorAndExit("Directory %s is not readable", dir)
	}

	if !fsutil.IsExecutable(dir) {
		printErrorAndExit("Directory %s is not executable", dir)
	}
}

// cloneRepository start repository clone process
func cloneRepository(url, dir string) {
	fmtc.Printfn("Fetching index from {*}%s{!}…", url)

	i, err := fetchIndex(url)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if i.Meta.Items == 0 {
		printErrorAndExit("Repository is empty")
	}

	printRepositoryInfo(i)

	uuid := getCurrentIndexUUID(dir)

	if uuid == i.UUID {
		fmtc.Println("{g}Looks like you already have the same set of data{!}")
		return
	}

	if !options.GetB(OPT_YES) {
		ok, err := input.ReadAnswer("Clone this repository?", "N")

		if err != nil || !ok {
			os.Exit(0)
		}
	}

	downloadRepositoryData(i, url, dir)
	saveIndex(i, dir)

	fmtc.NewLine()
	fmtc.Printfn("{g}Repository successfully cloned to {g*}%s{!}", dir)
}

// printRepositoryInfo prints basic info about repository data
func printRepositoryInfo(i *index.Index) {
	fmtutil.Separator(false, "REPOSITORY INFO")

	updated := timeutil.Format(time.Unix(i.Meta.Created, 0), "%Y/%m/%d %H:%M:%S")

	fmtc.Printfn("     {*}UUID{!}: %s", i.UUID)
	fmtc.Printfn("  {*}Updated{!}: %s\n", updated)

	for _, distName := range i.Data.Keys() {
		size, items := int64(0), 0
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

			fmtc.Printfn(
				"  {c*}%s{!}{c}/%s:{!} %3s {s-}|{!} %s", distName, archName,
				fmtutil.PrettyNum(items), fmtutil.PrettySize(size, " "),
			)
		}
	}

	fmtc.NewLine()
	fmtc.Printfn(
		"  {*}Total:{!} %s {s-}|{!} %s",
		fmtutil.PrettyNum(i.Meta.Items),
		fmtutil.PrettySize(i.Meta.Size, " "),
	)

	fmtutil.Separator(false)
}

// fetchIndex downloads remote repository index
func fetchIndex(url string) (*index.Index, error) {
	resp, err := req.Request{URL: url + "/" + INDEX_NAME}.Get()

	if err != nil {
		return nil, fmtc.Errorf("Can't fetch repository index: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmtc.Errorf("Can't fetch repository index: server return status code %d", resp.StatusCode)
	}

	repoIndex := &index.Index{}
	err = resp.JSON(repoIndex)

	if err != nil {
		return nil, fmtc.Errorf("Can't decode repository index: %v", err)
	}

	return repoIndex, nil
}

// downloadRepositoryData downloads all files from repository
func downloadRepositoryData(i *index.Index, url, dir string) {
	items := getItems(i, url)

	pb := progress.New(int64(len(items)), "Starting…")

	pbs := progress.DefaultSettings
	pbs.IsSize = false
	pbs.ShowSpeed = false
	pbs.ShowRemaining = false
	pbs.ShowName = false
	pbs.NameColorTag = "{*}"
	pbs.BarFgColorTag = colorTagApp
	pbs.PercentColorTag = ""
	pbs.RemainingColorTag = "{s}"

	pb.UpdateSettings(pbs)
	pb.Start()

	fmtc.Printfn(
		"Downloading %s %s from remote repository…",
		fmtutil.PrettyNum(len(items)),
		pluralize.Pluralize(len(items), "file", "files"),
	)

	for _, item := range items {
		fileDir := path.Join(dir, item.OS, item.Arch)
		filePath := path.Join(dir, item.OS, item.Arch, item.File)

		if !fsutil.IsExist(fileDir) {
			err := os.MkdirAll(fileDir, 0755)

			if err != nil {
				pb.Finish()
				fmtc.NewLine()
				printErrorAndExit("Can't create directory %s: %v", fileDir, err)
			}
		}

		if fsutil.IsExist(filePath) {
			fileSize := fsutil.GetSize(filePath)

			if fileSize == item.Size {
				pb.Add(1)
				continue
			}
		}

		err := downloadFile(item.URL, filePath)

		if err != nil {
			pb.Finish()
			fmtc.NewLine()
			printErrorAndExit("%v", err)
		}

		pb.Add(1)
	}

	pb.Finish()

	fmtc.Printfn("\n{g}Repository successfully cloned into %s{!}")
}

// getItems returns slice with info about items in repository
func getItems(repoIndex *index.Index, url string) []FileInfo {
	var items []FileInfo

	for _, os := range repoIndex.Data.Keys() {
		for _, arch := range repoIndex.Data[os].Keys() {
			for _, category := range repoIndex.Data[os][arch].Keys() {
				for _, version := range repoIndex.Data[os][arch][category] {
					items = append(items, FileInfo{
						File: version.File,
						URL:  url + "/" + version.Path + "/" + version.File,
						OS:   os,
						Arch: arch,
						Size: version.Size,
					})

					if len(version.Variations) != 0 {
						for _, subVersion := range version.Variations {
							items = append(items, FileInfo{
								File: subVersion.File,
								URL:  url + "/" + subVersion.Path + "/" + subVersion.File,
								OS:   os,
								Arch: arch,
								Size: subVersion.Size,
							})
						}
					}
				}
			}
		}
	}

	return items
}

// downloadFile downloads and saves remote file
func downloadFile(url, output string) error {
	if fsutil.IsExist(output) {
		os.Remove(output)
	}

	fd, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return fmtc.Errorf("Can't create file: %v", err)
	}

	defer fd.Close()

	resp, err := req.Request{URL: url}.Get()

	if err != nil {
		return fmtc.Errorf("Can't download file: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmtc.Errorf("Can't download file: server return status code %d", resp.StatusCode)
	}

	w := bufio.NewWriter(fd)
	_, err = io.Copy(w, resp.Body)

	w.Flush()

	if err != nil {
		return fmtc.Errorf("Can't write file: %v", err)
	}

	return nil
}

// saveIndex encodes index to JSON format and saves it into the file
func saveIndex(repoIndex *index.Index, dir string) {
	indexPath := path.Join(dir, INDEX_NAME)

	fmtc.Printf("Saving index… ")

	err := jsonutil.Write(indexPath, repoIndex)

	if err != nil {
		fmtc.Println("{r}ERROR{!}")
		printErrorAndExit("Can't save index as %s: %v", indexPath, err)
	}

	fmtc.Println("{g}DONE{!}")
}

// getCurrentIndexUUID returns current index UUID (if exist)
func getCurrentIndexUUID(dir string) string {
	indexFile := path.Join(dir, INDEX_NAME)

	if !fsutil.IsExist(indexFile) {
		return ""
	}

	i := &index.Index{}

	if jsonutil.Read(indexFile, i) != nil {
		return ""
	}

	return i.UUID
}

// printErrorAndExit print error message and exit with non-zero exit code
func printErrorAndExit(f string, a ...any) {
	terminal.Error(f, a...)
	fmtc.NewLine()
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
	fmt.Println(man.Generate(genUsage(), genAbout("")))
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo("", "url", "path")

	info.AppNameColorTag = colorTagApp

	info.AddOption(OPT_YES, `Answer "yes" to all questions`)
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"https://rbinstall.kaos.st /path/to/clone",
		"Clone EK repository to /path/to/clone",
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

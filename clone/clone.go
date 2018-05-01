package clone

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2018 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"io"
	"os"
	"runtime"
	"time"

	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/fmtutil"
	"pkg.re/essentialkaos/ek.v9/fsutil"
	"pkg.re/essentialkaos/ek.v9/httputil"
	"pkg.re/essentialkaos/ek.v9/jsonutil"
	"pkg.re/essentialkaos/ek.v9/options"
	"pkg.re/essentialkaos/ek.v9/path"
	"pkg.re/essentialkaos/ek.v9/req"
	"pkg.re/essentialkaos/ek.v9/terminal"
	"pkg.re/essentialkaos/ek.v9/timeutil"
	"pkg.re/essentialkaos/ek.v9/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "RBInstall Clone"
	VER  = "0.6.0"
	DESC = "Utility for cloning RBInstall repository"
)

// Options
const (
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

// FileInfo contains info about file with data
type FileInfo struct {
	File string
	URL  string
	OS   string
	Arch string
	Size int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:      {Type: options.BOOL, Alias: "ver"},
}

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

	if options.GetB(OPT_HELP) || len(args) != 2 {
		showUsage()
		return
	}

	fmtutil.SizeSeparator = " "

	req.SetUserAgent("RBI-Cloner", VER)

	url := args[0]
	dir := args[1]

	checkArguments(url, dir)
	cloneRepository(url, dir)
}

// checkArguments checks command line arguments
func checkArguments(url, dir string) {
	if !httputil.IsURL(url) {
		printErrorAndExit("\nUrl %s doesn't looks like valid url\n", url)
	}

	if !fsutil.IsExist(dir) {
		printErrorAndExit("\nDirectory %s does not exist\n", dir)
	}

	if !fsutil.IsDir(dir) {
		printErrorAndExit("\nTarget %s is not a directory\n", dir)
	}

	if !fsutil.IsReadable(dir) {
		printErrorAndExit("\nDirectory %s is not readable\n", dir)
	}

	if !fsutil.IsExecutable(dir) {
		printErrorAndExit("\nDirectory %s is not exectable\n", dir)
	}

	if !fsutil.IsEmptyDir(dir) {
		printWarn("\nDirectory %s is not empty", dir)
	}
}

// cloneRepository start repository clone process
func cloneRepository(url, dir string) {
	repoIndex, err := fetchIndex(url)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if repoIndex.Meta.Items == 0 {
		printErrorAndExit("Repository is empty")
	}

	printRepositoryInfo(repoIndex)

	ok, err := terminal.ReadAnswer("Clone this repository?", "y")

	if !ok || err != nil {
		fmtc.NewLine()
		os.Exit(0)
	}

	fmtc.NewLine()

	downloadRepositoryData(repoIndex, url, dir)

	fmtutil.Separator(false)

	saveIndex(repoIndex, dir)

	fmtc.Printf("\n{g}Repository successfully cloned to {g*}%s{!}\n\n", dir)
}

// printRepositoryInfo print basic info about repository data
func printRepositoryInfo(repoIndex *index.Index) {
	fmtutil.Separator(false, "REPOSITORY INFO")

	updatedDate := time.Unix(repoIndex.Meta.Created, 0)

	fmtc.Printf("  Updated: %s\n", timeutil.Format(updatedDate, "%Y/%m/%d %H:%M:%S"))
	fmtc.Printf("  Items: %s\n", fmtutil.PrettyNum(repoIndex.Meta.Items))
	fmtc.Printf("  Total Size: %s\n", fmtutil.PrettySize(repoIndex.Meta.Size))

	fmtutil.Separator(false)
}

// fetchIndex download remote repository index
func fetchIndex(url string) (*index.Index, error) {
	resp, err := req.Request{URL: url + "/index.json"}.Get()

	if err != nil {
		return nil, fmtc.Errorf("Can't fetch repository index: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmtc.Errorf("Can't fetch repository index: server return status code %d", resp.StatusCode)
	}

	repoIndex := index.NewIndex()

	err = resp.JSON(repoIndex)

	if err != nil {
		return nil, fmtc.Errorf("Can't decode repository index: %v", err)
	}

	return repoIndex, nil
}

// downloadRepositoryData download all files from repository
func downloadRepositoryData(repoIndex *index.Index, url, dir string) {
	items := getItems(repoIndex, url)

	fmtc.Printf("{s-}Downloading %d items...{!}\n\n", len(items))

	for _, item := range items {
		fileDir := path.Join(dir, item.OS, item.Arch)
		filePath := path.Join(dir, item.OS, item.Arch, item.File)

		if !fsutil.IsExist(fileDir) {
			err := os.MkdirAll(fileDir, 0755)

			if err != nil {
				printErrorAndExit("Can't create directory %s: %v", fileDir, err)
			}
		}

		if fsutil.IsExist(filePath) {
			fileSize := fsutil.GetSize(filePath)

			if fileSize == item.Size {
				fmtc.Printf(
					"{s-}  %-28s → %s/%s{!}\n",
					item.File, item.OS, item.Arch,
				)

				continue
			} else {
				os.Remove(filePath)
			}
		}

		fmtc.Printf(
			"{g}↓ %-28s{!} → {c}%s/%s{!} ",
			item.File, item.OS, item.Arch,
		)

		dlTime, err := downloadFile(item.URL, filePath)

		if err != nil {
			fmtc.Println("{r}ERROR{!}\n")
			printErrorAndExit("%v\n", err)
		}

		fmtc.Printf("{g}DONE{!} {s-}(%s){!}\n", timeutil.PrettyDuration(dlTime))
	}
}

// getItems return slice with info about items in repository
func getItems(repoIndex *index.Index, url string) []FileInfo {
	var items []FileInfo

	for osName, os := range repoIndex.Data {
		for archName, arch := range os {
			for _, category := range arch {
				for _, version := range category {
					items = append(items, FileInfo{
						File: version.File,
						URL:  url + "/" + version.Path + "/" + version.File,
						OS:   osName,
						Arch: archName,
						Size: version.Size,
					})

					if len(version.Variations) != 0 {
						for _, subVersion := range version.Variations {
							items = append(items, FileInfo{
								File: subVersion.File,
								URL:  url + "/" + subVersion.Path + "/" + subVersion.File,
								OS:   osName,
								Arch: archName,
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

// downloadFile download and save remote file
func downloadFile(url, output string) (time.Duration, error) {
	start := time.Now()

	if fsutil.IsExist(output) {
		os.Remove(output)
	}

	fd, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return 0, fmtc.Errorf("Can't create output file: %v", err)
	}

	defer fd.Close()

	resp, err := req.Request{URL: url}.Get()

	if err != nil {
		return time.Since(start), fmtc.Errorf("Can't download file: %v", err)
	}

	if resp.StatusCode != 200 {
		return time.Since(start), fmtc.Errorf("Can't download file: server return status code %d", resp.StatusCode)
	}

	_, err = io.Copy(fd, resp.Body)

	if err != nil {
		return time.Since(start), fmtc.Errorf("Can't save file: %v", err)
	}

	return time.Since(start), nil
}

// saveIndex encode index to json format and save to file
func saveIndex(repoIndex *index.Index, dir string) {
	indexPath := path.Join(dir, "index.json")

	fmtc.Printf("Saving index... ")

	err := jsonutil.EncodeToFile(indexPath, repoIndex)

	if err != nil {
		fmtc.Println("{r}ERROR{!}")
		printErrorAndExit("Can't save index as %s: %v", indexPath, err)
	}

	fmtc.Println("{g}DONE{!}")
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
	info := usage.NewInfo("", "url", "path")

	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"https://rbinstall.kaos.io /path/to/clone",
		"Clone EK repository to /path/to/clone",
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

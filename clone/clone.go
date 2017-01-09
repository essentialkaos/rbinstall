package clone

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"io"
	"os"
	"runtime"
	"time"

	"pkg.re/essentialkaos/ek.v6/arg"
	"pkg.re/essentialkaos/ek.v6/fmtc"
	"pkg.re/essentialkaos/ek.v6/fmtutil"
	"pkg.re/essentialkaos/ek.v6/fsutil"
	"pkg.re/essentialkaos/ek.v6/httputil"
	"pkg.re/essentialkaos/ek.v6/jsonutil"
	"pkg.re/essentialkaos/ek.v6/path"
	"pkg.re/essentialkaos/ek.v6/req"
	"pkg.re/essentialkaos/ek.v6/terminal"
	"pkg.re/essentialkaos/ek.v6/timeutil"
	"pkg.re/essentialkaos/ek.v6/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall Clone"
	VER  = "0.2.3"
	DESC = "Utility for cloning RBInstall repository"
)

const (
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

type FileInfo struct {
	File string
	URL  string
	OS   string
	Arch string
	Size int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

var argList = arg.Map{
	ARG_NO_COLOR: &arg.V{Type: arg.BOOL},
	ARG_HELP:     &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:      &arg.V{Type: arg.BOOL, Alias: "ver"},
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Init is main func
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

	if arg.GetB(ARG_HELP) || len(args) != 2 {
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
		printError("\nUrl %s doesn't looks like valid url\n", url)
		os.Exit(1)
	}

	if !fsutil.IsExist(dir) {
		printError("\nDirectory %s is not exist\n", dir)
		os.Exit(1)
	}

	if !fsutil.IsDir(dir) {
		printError("\nTarget %s is not a directory\n", dir)
		os.Exit(1)
	}

	if !fsutil.IsReadable(dir) {
		printError("\nDirectory %s is not readable\n", dir)
		os.Exit(1)
	}

	if !fsutil.IsExecutable(dir) {
		printError("\nDirectory %s is not exectable\n", dir)
		os.Exit(1)
	}

	if !fsutil.IsEmptyDir(dir) {
		printWarn("\nDirectory %s is not empty", dir)
	}
}

// cloneRepository start repository clone process
func cloneRepository(url, dir string) {
	repoIndex, err := fetchIndex(url)

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if repoIndex.Meta.Items == 0 {
		printWarn("Repository is empty")
		os.Exit(0)
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
				printError("Can't create directory %s: %v", fileDir, err)
				os.Exit(1)
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
			printError("%v\n", err)

			os.Exit(1)
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
		printError("Can't save index as %s: %v", indexPath, err)
		os.Exit(1)
	}

	fmtc.Println("{g}DONE{!}")
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
	info := usage.NewInfo("", "url", "path")

	info.AddOption(ARG_NO_COLOR, "Disable colors in output")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VER, "Show version")

	info.AddExample("https://rbinstall.kaos.io /path/to/clone", "Clone EK repository to /path/to/clone")

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

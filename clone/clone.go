package clone

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"io"
	"os"
	"runtime"
	"time"

	"pkg.re/essentialkaos/ek.v3/arg"
	"pkg.re/essentialkaos/ek.v3/fmtc"
	"pkg.re/essentialkaos/ek.v3/fmtutil"
	"pkg.re/essentialkaos/ek.v3/fsutil"
	"pkg.re/essentialkaos/ek.v3/httputil"
	"pkg.re/essentialkaos/ek.v3/jsonutil"
	"pkg.re/essentialkaos/ek.v3/path"
	"pkg.re/essentialkaos/ek.v3/req"
	"pkg.re/essentialkaos/ek.v3/terminal"
	"pkg.re/essentialkaos/ek.v3/timeutil"
	"pkg.re/essentialkaos/ek.v3/usage"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall Clone"
	VER  = "0.1.2"
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

	fileCount, totalSize := getRepoStats(repoIndex)

	fmtutil.Separator(false, "Repository Statistics")

	fmtc.Printf("  {*}Index:{!}   %s\n", url+"/index.json")
	fmtc.Printf("  {*}Files:{!}   %s\n", fmtutil.PrettyNum(fileCount))
	fmtc.Printf("  {*}Size: {!}   %s\n", fmtutil.PrettySize(totalSize))

	fmtutil.Separator(false)

	ok, err := terminal.ReadAnswer("Clone this repository?", "y")

	if !ok || err != nil {
		fmtc.NewLine()
		os.Exit(0)
	}

	fmtc.NewLine()

	downloadRepositoryData(repoIndex, url, dir)
	saveIndex(repoIndex, dir)

	fmtc.Printf("\n{g}Repository successfully cloned to %s{!}\n\n", dir)
}

// fetchIndex download remote repository index
func fetchIndex(url string) (*index.Index, error) {
	resp, err := req.Request{URL: url + "/index.json"}.Get()

	if err != nil {
		return nil, fmtc.Errorf("Can't fetch repo index: %v", err)
	}

	repoIndex := index.NewIndex()

	err = resp.JSON(repoIndex)

	if err != nil {
		return nil, fmtc.Errorf("Can't decode repo index json: %v", err)
	}

	return repoIndex, nil
}

// getRepoStats return number of files in repo and total size
func getRepoStats(repoIndex *index.Index) (int, int64) {
	var (
		fileCount int
		totalSize uint64
	)

	for _, archData := range repoIndex.Data {
		for _, categoryData := range archData {
			for _, versionData := range categoryData.Versions {
				totalSize += versionData.Size
				fileCount++
			}
		}
	}

	return fileCount, int64(totalSize)
}

// downloadRepositoryData download all files from repository
func downloadRepositoryData(repoIndex *index.Index, url, dir string) {
	fmtc.Println("Downloading repository data...\n")

	for arch, archData := range repoIndex.Data {
		os.Mkdir(path.Join(dir, arch), 0755)

		for category, categoryData := range archData {
			for _, versionData := range categoryData.Versions {
				fmtc.Printf("%-36s → ", arch+"/"+category+"/"+versionData.Name)

				fileURL := url + "/" + versionData.Path
				outputFile := path.Join(dir, versionData.Path)

				dlTime, err := downloadFile(fileURL, outputFile)

				if err != nil {
					fmtc.Println("{r}ERROR{!}\n")
					printError("%v\n", err)

					os.Exit(1)
				}

				fmtc.Printf("{g}DONE{!} {s}(%s){!}\n", timeutil.PrettyDuration(dlTime))
			}
		}
	}
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

	resp, err := req.Request{
		URL:       url,
		UserAgent: "RBI-Cloner/" + VER,
	}.Get()

	if err != nil {
		return time.Since(start), fmtc.Errorf("Can't download file: %v", err)
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

	fmtc.Printf("\nSaving index to %s... ", indexPath)

	err := jsonutil.EncodeToFile(indexPath, repoIndex)

	if err != nil {
		fmtc.Println("{r}ERROR{!}")
		printError("Can't save index to %s: %v", indexPath, err)
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

	info.AddExample("https://rbrepo.kaos.io /path/to/clone", "Clone EK repository to /path/to/clone")

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

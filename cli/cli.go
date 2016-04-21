package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2015 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/crypto"
	"pkg.re/essentialkaos/ek.v1/fmtc"
	"pkg.re/essentialkaos/ek.v1/fmtutil"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/knf"
	"pkg.re/essentialkaos/ek.v1/log"
	"pkg.re/essentialkaos/ek.v1/req"
	"pkg.re/essentialkaos/ek.v1/sortutil"
	"pkg.re/essentialkaos/ek.v1/system"
	"pkg.re/essentialkaos/ek.v1/tmp"
	"pkg.re/essentialkaos/ek.v1/usage"

	"pkg.re/essentialkaos/z7.v1"

	"github.com/cheggaaa/pb"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall"
	VER  = "0.5.0"
	DESC = "Utility for installing prebuilt ruby versions to RBEnv"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	ARG_GEMS_UPDATE   = "g:gems-update"
	ARG_GEMS_INSECURE = "S:gems-insecure"
	ARG_NO_COLOR      = "nc:no-color"
	ARG_HELP          = "h:help"
	ARG_VER           = "v:version"
)

const (
	MAIN_TMP_DIR          = "main:tmp-dir"
	STORAGE_URL           = "storage:url"
	RBENV_DIR             = "rbenv:dir"
	RBENV_ALLOW_OVERWRITE = "rbenv:allow-overwrite"
	GEMS_NO_RI            = "gems:no-ri"
	GEMS_NO_RDOC          = "gems:no-rdoc"
	GEMS_SOURCE           = "gems:source"
	GEMS_SOURCE_SECURE    = "gems:source-secure"
	GEMS_INSTALL          = "gems:install"
	LOG_FILE              = "log:file"
	LOG_PERMS             = "log:perms"
	LOG_LEVEL             = "log:level"
)

const (
	CATEGORY_RUBY     = "ruby"
	CATEGORY_JRUBY    = "jruby"
	CATEGORY_REE      = "ree"
	CATEGORY_RUBINIUS = "rubinius"
	CATEGORY_OTHER    = "other"
)

const CONFIG_FILE = "/etc/rbinstall.conf"
const FAIL_LOG_NAME = "rbinstall-fail.log"
const VERSION_FILE = ".ruby-version"

// ////////////////////////////////////////////////////////////////////////////////// //

type PassThru struct {
	io.Reader
	pb *pb.ProgressBar
}

type VersionInfo struct {
	Name         string `json:"name"`
	File         string `json:"file"`
	Path         string `json:"path"`
	Size         uint64 `json:"size"`
	Hash         string `json:"hash"`
	RailsExpress bool   `json:"rx"`
}

type CategoryInfo struct {
	Versions []*VersionInfo `json:"versions"`
}

type Index struct {
	Data map[string]map[string]*CategoryInfo `json:"data"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_GEMS_UPDATE:   &arg.V{Type: arg.BOOL},
	ARG_GEMS_INSECURE: &arg.V{Type: arg.BOOL},
	ARG_NO_COLOR:      &arg.V{Type: arg.BOOL},
	ARG_HELP:          &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:           &arg.V{Type: arg.BOOL, Alias: "ver"},
}

var (
	index       *Index
	temp        *tmp.Temp
	currentUser *system.User
	runDate     time.Time
)

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	var errs []error

	args, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		fmtc.NewLine()

		for _, err := range errs {
			fmtc.Printf("{r}%v{!}\n", err)
		}

		exit(1)
	}

	if arg.GetB(ARG_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if arg.GetB(ARG_VER) {
		showAbout()
		return
	}

	if arg.GetB(ARG_HELP) {
		showUsage()
		return
	}

	fmtc.NewLine()

	prepare()
	fetchIndex()

	if len(args) != 0 {
		checkPerms()
		setupLogger()
		setupTemp()

		if arg.GetB(ARG_GEMS_UPDATE) {
			updateGems(args[0])
		} else {
			installCommand(args[0])
		}
	} else {
		if fsutil.CheckPerms("FRS", VERSION_FILE) {
			installFromVersionFile()
		} else {
			listCommand()
		}
	}

	exit(0)
}

// prepare do some preparations
func prepare() {
	req.UserAgent = fmt.Sprintf("%s/%s (go; %s; %s-%s)",
		APP, VER, runtime.Version(),
		runtime.GOARCH, runtime.GOOS)

	loadConfig()
	validateConfig()
}

// checkPerms check user for sudo
func checkPerms() {
	var err error

	currentUser, err = system.CurrentUser()

	if err != nil {
		fmtc.Printf("{r}%v{!}\n", err)
		exit(1)
	}

	if !currentUser.IsRoot() {
		fmtc.Println("{r}This action requires superuser (root) privileges{!}")
		exit(1)
	}
}

// setupLogger setup logging subsystem
func setupLogger() {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS))

	if err != nil {
		fmtc.Printf("{r}%s{!}\n", err.Error())
		exit(1)
	}

	log.MinLevel(knf.GetI(LOG_LEVEL))
}

// setupTemp setup dir for temporary data
func setupTemp() {
	var err error

	temp, err = tmp.NewTemp(knf.GetS(MAIN_TMP_DIR, "/tmp"))

	if err != nil {
		fmtc.Printf("{r}%v{!}\n", err)
		exit(1)
	}
}

// loadConfig load global config
func loadConfig() {
	err := knf.Global(CONFIG_FILE)

	if err != nil {
		fmtc.Printf("{r}%v{!}\n", err)
		exit(1)
	}
}

// validateConfig validate knf.values
func validateConfig() {
	var permsChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !fsutil.CheckPerms(value.(string), config.GetS(prop)) {
			switch value.(string) {
			case "DWX":
				return errors.New(fmt.Sprintf("Property %s must be path to writable directory.", prop))
			}
		}

		return nil
	}

	errs := knf.Validate([]*knf.Validator{
		&knf.Validator{MAIN_TMP_DIR, permsChecker, "DWX"},
		&knf.Validator{RBENV_DIR, permsChecker, "DWX"},
		&knf.Validator{STORAGE_URL, knf.Empty, nil},
	})

	if len(errs) != 0 {
		fmtc.Println("{r}Error while knf.validation:{!}")

		for _, err := range errs {
			fmtc.Printf("  {r}%s{!}\n", err.Error())
		}

		exit(1)
	}
}

// fetchIndex download index from remote repository
func fetchIndex() {
	resp, err := req.Request{URL: knf.GetS(STORAGE_URL) + "/index.json"}.Do()

	if err != nil {
		fmtc.Printf("{r}Can't fetch repo index: %s{!}\n", err.Error())
		exit(1)
	}

	index = &Index{}

	err = resp.JSON(index)

	if err != nil {
		fmtc.Printf("{r}Can't decode repo index json: %s{!}\n", err.Error())
		exit(1)
	}
}

func installFromVersionFile() {
	blob, err := ioutil.ReadFile(VERSION_FILE)
	if err != nil {
		fmtc.Println("Cannot read %s", VERSION_FILE)
		exit(1)
	}

	version, err := readVersionFromFile(string(blob))
	if err != nil {
		fmtc.Println("Cannot find version in %s", VERSION_FILE)
		exit(1)
	}

	fmtc.Println("Installing version %s from %s", version, VERSION_FILE)
	installCommand(version)
}

func readVersionFromFile(body string) (string, error) {
	matches := regexp.MustCompile(`^\s*(\S+)\s*`).FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", errors.New("No version in file")
	}
	return matches[1], nil
}

// listCommand show list of all available versions
func listCommand() {
	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		fmtc.Println("{y}Can't get information about system{!}")
		exit(1)
	}

	if index.Data[systemInfo.Arch] == nil {
		fmtc.Printf("{y}Prebuilt rubies not found for %s architecture{!}", systemInfo.Arch)
		exit(0)
	}

	fmtc.Printf("{dY*} %-26s{!} {dC*} %-26s{!} {dG*} %-26s{!} {dM*} %-26s{!} {dS*} %-26s{!}\n\n",
		strings.ToUpper(CATEGORY_RUBY),
		strings.ToUpper(CATEGORY_JRUBY),
		strings.ToUpper(CATEGORY_REE),
		strings.ToUpper(CATEGORY_RUBINIUS),
		strings.ToUpper(CATEGORY_OTHER),
	)

	var count int = 0

	var (
		ruby     []string = getVersionNames(index.Data[systemInfo.Arch][CATEGORY_RUBY])
		jruby    []string = getVersionNames(index.Data[systemInfo.Arch][CATEGORY_JRUBY])
		ree      []string = getVersionNames(index.Data[systemInfo.Arch][CATEGORY_REE])
		rubinius []string = getVersionNames(index.Data[systemInfo.Arch][CATEGORY_RUBINIUS])
		other    []string = getVersionNames(index.Data[systemInfo.Arch][CATEGORY_OTHER])
	)

	for {
		hasItems := false

		if ruby != nil && len(ruby) > count {
			switch {
			case strings.Contains(ruby[count], "-railsexpress"):
				baseName := strings.Replace(ruby[count], "-railsexpress", "", -1)
				fmtc.Printf(" %s{s}-railsexpress{!} "+getAlignSpaces(baseName+"-railsexpress", 26), baseName)
			case ruby[count] == "-none-":
				fmtc.Printf(" {s}%-26s{!} ", "-none-")
			default:
				fmtc.Printf(" %-26s ", ruby[count])
			}

			hasItems = true
		} else {
			fmtc.Printf(" %-26s ", "")
		}

		if jruby != nil && len(jruby) > count {
			switch {
			case jruby[count] == "-none-":
				fmtc.Printf(" {s}%-26s{!} ", "-none-")
			default:
				fmtc.Printf(" %-26s ", jruby[count])
			}

			hasItems = true
		} else {
			fmtc.Printf(" %-26s ", "")
		}

		if ree != nil && len(ree) > count {
			switch {
			case ree[count] == "-none-":
				fmtc.Printf(" {s}%-26s{!} ", "-none-")
			default:
				fmtc.Printf(" %-26s ", ree[count])
			}

			hasItems = true
		} else {
			fmtc.Printf(" %-26s ", "")
		}

		if rubinius != nil && len(rubinius) > count {
			switch {
			case rubinius[count] == "-none-":
				fmtc.Printf(" {s}%-26s{!} ", "-none-")
			default:
				fmtc.Printf(" %-26s ", rubinius[count])
			}

			hasItems = true
		} else {
			fmtc.Printf(" %-26s ", "")
		}

		if other != nil && len(other) > count {
			switch {
			case other[count] == "-none-":
				fmtc.Printf(" {s}%-26s{!} ", "-none-")
			default:
				fmtc.Printf(" %-26s ", other[count])
			}

			hasItems = true
		} else {
			fmtc.Printf(" %-26s", "")
		}

		if hasItems == false {
			break
		}

		fmtc.NewLine()

		count++
	}
}

// installCommand install some version of ruby
func installCommand(name string) {
	info := index.Find(name)

	if info == nil {
		fmtc.Printf("{y}Can't find info about version %s{!}\n", name)
		exit(1)
	}

	fullPath := getVersionPath(info.Name)

	if fsutil.IsExist(fullPath) {
		if knf.GetB(RBENV_ALLOW_OVERWRITE) {
			os.RemoveAll(fullPath)
		} else {
			fmtc.Printf("{y}Version %s already installed{!}\n", name)
			exit(1)
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	fmtc.Printf("Fetching {c}%s{!} {s}(%s){!}...\n", info.Name, fmtutil.PrettySize(info.Size))

	file, err := downloadFile(knf.GetS(STORAGE_URL)+info.Path, info.File)

	if err != nil {
		fmtc.Printf("{r}%s{!}\n", err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	checkHashTask := &Task{
		Desc:    "Checking sha1 checksum",
		Handler: checkHashTaskHandler,
	}

	_, err = checkHashTask.Start(file, info.Hash)

	if err != nil {
		fmtc.Printf("{r}%s{!}\n", err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	unpackTask := &Task{
		Desc:    "Unpacking 7z archive",
		Handler: unpackTaskHandler,
	}

	_, err = unpackTask.Start(file, knf.GetS(RBENV_DIR)+"/versions/")

	if err != nil {
		fmtc.Printf("{r}%s{!}\n", err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetS(GEMS_INSTALL) != "" {
		for _, gem := range strings.Split(knf.GetS(GEMS_INSTALL), " ") {
			gemInstallTask := &Task{
				Desc:    fmt.Sprintf("Installing %s", gem),
				Handler: installGemTaskHandler,
			}

			_, err := gemInstallTask.Start(info.Name, gem)

			if err != nil {
				fmtc.Printf("{r}%s{!}\n", err.Error())
				exit(1)
			}
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	rehashTask := &Task{
		Desc:    "Rehashing",
		Handler: rehashTaskHandler,
	}

	_, err = rehashTask.Start()

	if err != nil {
		fmtc.Printf("{r}%s{!}\n", err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	log.Info("[%s] %s %s", currentUser.RealName, "Installed version", info.Name)

	fmtc.NewLine()
	fmtc.Printf("{g}Version %s successfully installed!{!}\n", info.Name)
}

func checkHashTaskHandler(args ...string) (string, error) {
	file := args[0]
	hash := args[1]

	fileHash := crypto.FileHash(file)

	if hash != fileHash {
		return "", fmt.Errorf("Wrong file hash %s ≠ %s", hash, fileHash)
	}

	return "", nil
}

func unpackTaskHandler(args ...string) (string, error) {
	file := args[0]
	output := args[1]

	output, err := z7.Extract(&z7.Props{File: file, Output: output})

	if err != nil {
		actionLog, err := logFailedAction([]byte(output))

		return "", fmt.Errorf("7za return error: %s (7za output saved as %s)", err.Error(), actionLog)
	}

	return "", nil
}

func installGemTaskHandler(args ...string) (string, error) {
	version := args[0]
	gem := args[1]

	return runGemCmd(version, "install", gem)
}

func updateGemTaskHandler(args ...string) (string, error) {
	version := args[0]
	gem := args[1]

	return runGemCmd(version, "update", gem)
}

func rehashTaskHandler(args ...string) (string, error) {
	rehashCmd := exec.Command("rbenv", "rehash")
	return "", rehashCmd.Run()
}

// updateGems update gems installed by rbinstall on defined version
func updateGems(rubyVersion string) {
	fullPath := getVersionPath(rubyVersion)

	if !fsutil.IsExist(fullPath) {
		fmtc.Printf("{r}Version %s is not installed{!}\n", rubyVersion)
		exit(1)
	}

	runDate = time.Now()

	fmtc.Printf("Updating gems for {c}%s{!}...\n\n", rubyVersion)

	// //////////////////////////////////////////////////////////////////////////////// //

	for _, gem := range strings.Split(knf.GetS(GEMS_INSTALL), " ") {
		var updateGemTask *Task

		if isGemInstalled(rubyVersion, gem) {
			updateGemTask = &Task{
				Desc:    fmt.Sprintf("Updating %s", gem),
				Handler: updateGemTaskHandler,
			}
		} else {
			updateGemTask = &Task{
				Desc:    fmt.Sprintf("Installing %s", gem),
				Handler: installGemTaskHandler,
			}
		}

		installedVersion, err := updateGemTask.Start(rubyVersion, gem)

		if err == nil {
			if installedVersion != "" {
				log.Info(
					"[%s] Updated gem %s to version %s for %s",
					currentUser.RealName, gem, installedVersion, rubyVersion,
				)
			}
		} else {
			fmtc.Printf("{r}%s{!}\n", err.Error())
			exit(1)
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	rehashTask := &Task{
		Desc:    "Rehashing",
		Handler: rehashTaskHandler,
	}

	_, err := rehashTask.Start()

	if err != nil {
		fmtc.Printf("{r}%s{!}\n", err.Error())
		exit(1)
	}

	fmtc.NewLine()
	fmtc.Println("{g}All gems successfully updated!{!}")
}

// runGemCmd run some gem command for some version
func runGemCmd(rubyVersion, cmd, gem string) (string, error) {
	start := time.Now()
	path := getVersionPath(rubyVersion)
	gemCmd := exec.Command(path+"/bin/ruby", path+"/bin/gem", cmd, gem)

	if knf.GetB(GEMS_NO_RI) {
		gemCmd.Args = append(gemCmd.Args, "--no-ri")
	}

	if knf.GetB(GEMS_NO_RDOC) {
		gemCmd.Args = append(gemCmd.Args, "--no-rdoc")
	}

	if knf.GetS(GEMS_SOURCE) != "" {
		gemCmd.Args = append(gemCmd.Args, "--source", getGemSourceURL())
	}

	output, err := gemCmd.Output()

	if err == nil {
		version := getInstalledGemVersion(rubyVersion, gem, start)

		if version == "" {
			return "", nil
		}

		return version, nil
	}

	actionLog, err := logFailedAction(output)

	if err == nil {
		switch cmd {
		case "update":
			return "", fmt.Errorf("Can't update gem %s. Gem command output saved as %s", gem, actionLog)
		default:
			return "", fmt.Errorf("Can't install gem %s. Gem command output saved as %s", gem, actionLog)
		}

	}

	return "", nil
}

// downloadFile download file from remote host
func downloadFile(url, fileName string) (string, error) {
	tmpDir, err := temp.MkDir()

	if err != nil {
		return "", err
	}

	fd, err := os.OpenFile(tmpDir+"/"+fileName, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return "", err
	}

	defer fd.Close()

	resp, err := req.Request{URL: url}.Do()

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("Server return error code %d", resp.StatusCode))
	}

	bar := pb.New64(resp.ContentLength)

	bar.ShowCounters = false
	bar.Format("###  ")
	bar.SetMaxWidth(80)
	bar.SetRefreshRate(50 * time.Millisecond)

	defer bar.Finish()

	pt := &PassThru{
		Reader: resp.Body,
		pb:     bar.Start(),
	}

	_, err = io.Copy(fd, pt)

	return tmpDir + "/" + fileName, err
}

// getVersionNames return sorted version names
func getVersionNames(category *CategoryInfo) []string {
	result := make([]string, 0)

	if category == nil {
		result = append(result, "-none-")
		return result
	}

	for _, info := range category.Versions {
		if strings.Contains(info.Name, "-railsexpress") {
			continue
		}

		if info.RailsExpress {
			result = append(result, info.Name+"-railsexpress")
		} else {
			result = append(result, info.Name)
		}
	}

	sortutil.Versions(result)

	return result
}

// installedGemVersion return version of installed gem
func getInstalledGemVersion(rubyVersion string, gemName string, since time.Time) string {
	gemsDir := getVersionGemDirPath(rubyVersion)

	if gemsDir == "" {
		return ""
	}

	gemNameSize := len(gemName)

	for _, gem := range fsutil.List(gemsDir, true) {
		if len(gem) <= gemNameSize+1 {
			continue
		}

		if gem[0:gemNameSize+1] == gemName+"-" {
			modTime, _ := fsutil.GetMTime(gemsDir + "/" + gem)

			if modTime.Unix() > runDate.Unix() {
				return gem[gemNameSize+1:]
			}
		}
	}

	return ""
}

// isGemInstalled return true if given gem installed for given version
func isGemInstalled(rubyVersion string, gemName string) bool {
	gemsDir := getVersionGemDirPath(rubyVersion)

	if gemsDir == "" {
		return false
	}

	gemNameSize := len(gemName)

	for _, gem := range fsutil.List(gemsDir, true) {
		if len(gem) <= gemNameSize+1 {
			continue
		}

		if gem[0:gemNameSize+1] == gemName+"-" {
			return true
		}
	}

	return false
}

// getVersionGemPath return path to directory with installed gems
func getVersionGemDirPath(rubyVersion string) string {
	gemsPath := getVersionPath(rubyVersion) + "/lib/ruby/gems"

	if !fsutil.IsExist(gemsPath) {
		return ""
	}

	gemsDirList := fsutil.List(gemsPath, true)

	if len(gemsDirList) == 0 {
		return ""
	}

	return gemsPath + "/" + gemsDirList[0] + "/gems"
}

// getVersionPath return full path to directory for given ruby version
func getVersionPath(rubyVersion string) string {
	return knf.GetS(RBENV_DIR) + "/versions/" + rubyVersion
}

// getAlignSpaces return spaces for message align
func getAlignSpaces(t string, l int) string {
	spaces := "                                    "
	return spaces[0 : l-len(t)]
}

func getGemSourceURL() string {
	if !arg.GetB(ARG_GEMS_INSECURE) && knf.GetB(GEMS_SOURCE_SECURE, false) {
		return "https://" + knf.GetS(GEMS_SOURCE)
	}

	return "http://" + knf.GetS(GEMS_SOURCE)
}

// logFailedAction save data to temporary log file and return path
// to this log file
func logFailedAction(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("Output data is empty")
	}

	tmpName := knf.GetS(MAIN_TMP_DIR) + "/" + FAIL_LOG_NAME

	if fsutil.IsExist(tmpName) {
		os.Remove(tmpName)
	}

	data = append(data, []byte("\n\n")...)

	err := ioutil.WriteFile(tmpName, data, 0666)

	if err != nil {
		return "", err
	}

	return tmpName, nil
}

// exit exits clean temporary data and exit from utility with given exit code
func exit(code int) {
	if temp != nil {
		temp.Clean()
	}

	fmtc.NewLine()
	os.Exit(code)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Find info by name
func (i *Index) Find(name string) *VersionInfo {
	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		return nil
	}

	for _, category := range i.Data[systemInfo.Arch] {
		info := category.Find(name)

		if info != nil {
			return info
		}
	}

	return nil
}

// Find info by name
func (ci *CategoryInfo) Find(name string) *VersionInfo {
	if len(ci.Versions) == 0 {
		return nil
	}

	for _, info := range ci.Versions {
		if info.Name == name {
			return info
		}
	}

	return nil
}

// Read proxy for progress bar
func (pt *PassThru) Read(p []byte) (int, error) {
	n, err := pt.Reader.Read(p)

	if n > 0 {
		pt.pb.Add64(int64(n))
	}

	return n, err
}

// ////////////////////////////////////////////////////////////////////////////////// //

func showUsage() {
	info := usage.NewInfo("", "version")

	info.AddOption(ARG_GEMS_UPDATE, "Update gems for some version")
	info.AddOption(ARG_GEMS_INSECURE, "Use http instead https for installing gems")
	info.AddOption(ARG_NO_COLOR, "Disable colors in output")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VER, "Show version")

	info.AddExample("2.0.0-p598", "Install 2.0.0-p598")
	info.AddExample("2.0.0-p598-railsexpress", "Install 2.0.0-p598 with railsexpress patches")
	info.AddExample("2.0.0-p598 -g", "Update gems installed on 2.0.0-p598")

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

package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
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
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/crypto"
	"pkg.re/essentialkaos/ek.v1/env"
	"pkg.re/essentialkaos/ek.v1/fmtc"
	"pkg.re/essentialkaos/ek.v1/fmtutil"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/knf"
	"pkg.re/essentialkaos/ek.v1/log"
	"pkg.re/essentialkaos/ek.v1/req"
	"pkg.re/essentialkaos/ek.v1/signal"
	"pkg.re/essentialkaos/ek.v1/system"
	"pkg.re/essentialkaos/ek.v1/terminal"
	"pkg.re/essentialkaos/ek.v1/tmp"
	"pkg.re/essentialkaos/ek.v1/usage"

	"pkg.re/essentialkaos/z7.v2"

	"github.com/cheggaaa/pb"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RBInstall"
	VER  = "0.7.4"
	DESC = "Utility for installing prebuilt ruby versions to RBEnv"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// List of supported command-line arguments
const (
	ARG_GEMS_UPDATE   = "g:gems-update"
	ARG_GEMS_INSECURE = "S:gems-insecure"
	ARG_RUBY_VERSION  = "r:ruby-version"
	ARG_NO_COLOR      = "nc:no-color"
	ARG_NO_PROGRESS   = "np:no-progress"
	ARG_HELP          = "h:help"
	ARG_VER           = "v:version"
)

// List of supported config values
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

// List of default ruby categories
const (
	CATEGORY_RUBY     = "ruby"
	CATEGORY_JRUBY    = "jruby"
	CATEGORY_REE      = "ree"
	CATEGORY_RUBINIUS = "rubinius"
	CATEGORY_OTHER    = "other"
)

// Path to config file
const CONFIG_FILE = "/etc/rbinstall.conf"

// Name of log with failed actions (gem install)
const FAIL_LOG_NAME = "rbinstall-fail.log"

// Value for column without any versions
const NONE_VERSION = "- none -"

// Default category column size
const DEFAULT_CATEGORY_SIZE = 28

// ////////////////////////////////////////////////////////////////////////////////// //

type PassThru struct {
	io.Reader
	pb *pb.ProgressBar
}

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_GEMS_UPDATE:   &arg.V{Type: arg.BOOL},
	ARG_GEMS_INSECURE: &arg.V{Type: arg.BOOL},
	ARG_RUBY_VERSION:  &arg.V{Type: arg.BOOL},
	ARG_NO_COLOR:      &arg.V{Type: arg.BOOL},
	ARG_NO_PROGRESS:   &arg.V{Type: arg.BOOL},
	ARG_HELP:          &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:           &arg.V{Type: arg.BOOL, Alias: "ver"},
}

var (
	repoIndex   *index.Index
	temp        *tmp.Temp
	currentUser *system.User
	runDate     time.Time
)

var categoryColor = map[string]string{
	CATEGORY_RUBY:     "y",
	CATEGORY_JRUBY:    "c",
	CATEGORY_REE:      "g",
	CATEGORY_RUBINIUS: "m",
	CATEGORY_OTHER:    "s",
}

var categorySize = map[string]int{}

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	var err error
	var errs []error

	runtime.GOMAXPROCS(2)

	args, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		fmtc.NewLine()

		for _, err := range errs {
			printError(err.Error())
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

	var rubyVersion string

	if len(args) != 0 {
		rubyVersion = args[0]
	} else if arg.GetB(ARG_RUBY_VERSION) {
		rubyVersion, err = getVersionFromFile()

		if err != nil {
			printError(err.Error())
			exit(1)
		}

		fmtc.Printf("{s}Installing version %s from version file{!}\n\n", rubyVersion)
	}

	if rubyVersion != "" {
		checkPerms()
		setupLogger()
		setupTemp()

		if arg.GetB(ARG_GEMS_UPDATE) {
			updateGems(rubyVersion)
		} else {
			installCommand(rubyVersion)
		}
	} else {
		listCommand()
	}

	exit(0)
}

// prepare do some preparations for installing ruby
func prepare() {
	req.UserAgent = fmtc.Sprintf("%s/%s (go; %s; %s-%s)",
		APP, VER, runtime.Version(),
		runtime.GOARCH, runtime.GOOS)

	loadConfig()
	validateConfig()

	signal.Handlers{
		signal.INT: intSignalHandler,
	}.TrackAsync()
}

// checkPerms check user for sudo
func checkPerms() {
	var err error

	currentUser, err = system.CurrentUser()

	if err != nil {
		printError(err.Error())
		exit(1)
	}

	if !currentUser.IsRoot() {
		printError("This action requires superuser (root) privileges")
		exit(1)
	}
}

// setupLogger setup logging subsystem
func setupLogger() {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS))

	if err != nil {
		printError(err.Error())
		exit(1)
	}

	log.MinLevel(knf.GetI(LOG_LEVEL))
}

// setupTemp setup dir for temporary data
func setupTemp() {
	var err error

	temp, err = tmp.NewTemp(knf.GetS(MAIN_TMP_DIR, "/tmp"))

	if err != nil {
		printError(err.Error())
		exit(1)
	}
}

// loadConfig load global config
func loadConfig() {
	err := knf.Global(CONFIG_FILE)

	if err != nil {
		printError(err.Error())
		exit(1)
	}
}

// validateConfig validate knf.values
func validateConfig() {
	var permsChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !fsutil.CheckPerms(value.(string), config.GetS(prop)) {
			switch value.(string) {
			case "DWX":
				return fmtc.Errorf("Property %s must be path to writable directory.", prop)
			}
		}

		return nil
	}

	errs := knf.Validate([]*knf.Validator{
		&knf.Validator{MAIN_TMP_DIR, permsChecker, "DWX"},
		&knf.Validator{STORAGE_URL, knf.Empty, nil},
	})

	if len(errs) != 0 {
		printError("Error while knf.validation:")

		for _, err := range errs {
			printError("  %v", err)
		}

		exit(1)
	}
}

// fetchIndex download index from remote repository
func fetchIndex() {
	resp, err := req.Request{URL: knf.GetS(STORAGE_URL) + "/index.json"}.Do()

	if err != nil {
		printError("Can't fetch repo index: %v", err)
		exit(1)
	}

	repoIndex = index.NewIndex()

	err = resp.JSON(repoIndex)

	if err != nil {
		printError("Can't decode repo index json: %v", err)
		exit(1)
	}

	repoIndex.Sort()
}

// listCommand show list of all available versions
func listCommand() {
	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		printWarn("Can't get information about system")
		exit(1)
	}

	if repoIndex.Data[systemInfo.Arch] == nil {
		printWarn("Prebuilt rubies not found for %s architecture", systemInfo.Arch)
		exit(0)
	}

	var (
		ruby     = getVersionNames(repoIndex.Data[systemInfo.Arch][CATEGORY_RUBY])
		jruby    = getVersionNames(repoIndex.Data[systemInfo.Arch][CATEGORY_JRUBY])
		ree      = getVersionNames(repoIndex.Data[systemInfo.Arch][CATEGORY_REE])
		rubinius = getVersionNames(repoIndex.Data[systemInfo.Arch][CATEGORY_RUBINIUS])
		other    = getVersionNames(repoIndex.Data[systemInfo.Arch][CATEGORY_OTHER])

		installed = getInstalledVersionsMap()
	)

	configureCategorySizes(map[string][]string{
		CATEGORY_RUBY:     ruby,
		CATEGORY_JRUBY:    jruby,
		CATEGORY_REE:      ree,
		CATEGORY_RUBINIUS: rubinius,
		CATEGORY_OTHER:    other,
	})

	var index int

	headerTemplate := fmt.Sprintf(
		"{dY} %%-%ds{!} {dC} %%-%ds{!} {dG} %%-%ds{!} {dM} %%-%ds{!} {dS} %%-%ds{!}\n\n",
		categorySize[CATEGORY_RUBY], categorySize[CATEGORY_JRUBY],
		categorySize[CATEGORY_REE], categorySize[CATEGORY_RUBINIUS],
		categorySize[CATEGORY_OTHER],
	)

	fmtc.Printf(
		headerTemplate,
		strings.ToUpper(CATEGORY_RUBY),
		strings.ToUpper(CATEGORY_JRUBY),
		strings.ToUpper(CATEGORY_REE),
		strings.ToUpper(CATEGORY_RUBINIUS),
		strings.ToUpper(CATEGORY_OTHER),
	)

	for {

		hasItems := false

		hasItems = printCurrentVersionName(CATEGORY_RUBY, ruby, installed, index) || hasItems
		hasItems = printCurrentVersionName(CATEGORY_JRUBY, jruby, installed, index) || hasItems
		hasItems = printCurrentVersionName(CATEGORY_REE, ree, installed, index) || hasItems
		hasItems = printCurrentVersionName(CATEGORY_RUBINIUS, rubinius, installed, index) || hasItems
		hasItems = printCurrentVersionName(CATEGORY_OTHER, other, installed, index) || hasItems

		if !hasItems {
			break
		}

		fmtc.NewLine()

		index++
	}
}

// installCommand install some version of ruby
func installCommand(rubyVersion string) {
	info, category := repoIndex.Find(rubyVersion)

	if info == nil {
		printWarn("Can't find info about version %s", rubyVersion)
		exit(1)
	}

	checkRBEnvDirPerms()
	checkDependencies(category)

	if isVersionInstalled(info.Name) {
		if knf.GetB(RBENV_ALLOW_OVERWRITE) {
			os.RemoveAll(getVersionPath(info.Name))
		} else {
			printWarn("Version %s already installed", info.Name)
			exit(0)
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	fmtc.Printf("Fetching {c}%s{!} {s}(%s){!}...\n", info.Name, fmtutil.PrettySize(info.Size))

	file, err := downloadFile(knf.GetS(STORAGE_URL)+info.Path, info.File)

	if err != nil {
		printError(err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	checkHashTask := &Task{
		Desc:    "Checking sha1 checksum",
		Handler: checkHashTaskHandler,
	}

	_, err = checkHashTask.Start(file, info.Hash)

	if err != nil {
		printError(err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	unpackTask := &Task{
		Desc:    "Unpacking 7z archive",
		Handler: unpackTaskHandler,
	}

	_, err = unpackTask.Start(file, getRBEnvVersionsPath())

	if err != nil {
		printError(err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetS(GEMS_INSTALL) != "" {
		for _, gem := range strings.Split(knf.GetS(GEMS_INSTALL), " ") {
			gemInstallTask := &Task{
				Desc:    fmtc.Sprintf("Installing %s", gem),
				Handler: installGemTaskHandler,
			}

			_, err = gemInstallTask.Start(info.Name, gem)

			if err != nil {
				printError(err.Error())
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
		printError(err.Error())
		exit(1)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	log.Info("[%s] %s %s", currentUser.RealName, "Installed version", info.Name)

	fmtc.NewLine()
	fmtc.Printf("{g}Version %s successfully installed!{!}\n", info.Name)
}

// getVersionFromFile try to read version file and return defined version
func getVersionFromFile() (string, error) {
	versionFile := fsutil.ProperPath("FRS",
		[]string{
			".ruby-version",
			".rbenv-version",
		},
	)

	if versionFile == "" {
		return "", fmtc.Errorf("Can't find proper version file")
	}

	versionData, err := ioutil.ReadFile(versionFile)

	if err != nil {
		return "", fmtc.Errorf("Can't read version file: %v", err)
	}

	versionName := strings.Trim(string(versionData[:]), " \n\r")

	if versionName == "" {
		return "", fmtc.Errorf("Can't use version file - file malformed")
	}

	return versionName, nil
}

func checkHashTaskHandler(args ...string) (string, error) {
	file := args[0]
	hash := args[1]

	fileHash := crypto.FileHash(file)

	if hash != fileHash {
		return "", fmtc.Errorf("Wrong file hash %s ≠ %s", hash, fileHash)
	}

	return "", nil
}

func unpackTaskHandler(args ...string) (string, error) {
	file := args[0]
	outputDir := args[1]

	output, err := z7.Extract(&z7.Props{File: file, OutputDir: outputDir})

	if err != nil {
		actionLog, err := logFailedAction([]byte(output))

		return "", fmtc.Errorf("7za return error: %s (7za output saved as %s)", err.Error(), actionLog)
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
		printError("Version %s is not installed", rubyVersion)
		exit(1)
	}

	checkRBEnvDirPerms()

	runDate = time.Now()

	fmtc.Printf("Updating gems for {c}%s{!}...\n\n", rubyVersion)

	// //////////////////////////////////////////////////////////////////////////////// //

	for _, gem := range strings.Split(knf.GetS(GEMS_INSTALL), " ") {
		var updateGemTask *Task

		if isGemInstalled(rubyVersion, gem) {
			updateGemTask = &Task{
				Desc:    fmtc.Sprintf("Updating %s", gem),
				Handler: updateGemTaskHandler,
			}
		} else {
			updateGemTask = &Task{
				Desc:    fmtc.Sprintf("Installing %s", gem),
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
			printError(err.Error())
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
		printError(err.Error())
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
			return "", fmtc.Errorf("Can't update gem %s. Gem command output saved as %s", gem, actionLog)
		default:
			return "", fmtc.Errorf("Can't install gem %s. Gem command output saved as %s", gem, actionLog)
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
		return "", fmtc.Errorf("Server return error code %d", resp.StatusCode)
	}

	if arg.GetB(ARG_NO_PROGRESS) {
		_, err = io.Copy(fd, resp.Body)
	} else {
		bar := pb.New64(resp.ContentLength)

		bar.ShowCounters = false
		bar.Format("——→  ")
		bar.SetMaxWidth(80)
		bar.SetRefreshRate(50 * time.Millisecond)

		defer bar.Finish()

		pt := &PassThru{
			Reader: resp.Body,
			pb:     bar.Start(),
		}

		_, err = io.Copy(fd, pt)
	}

	return tmpDir + "/" + fileName, err
}

// getVersionNames return sorted version names
func getVersionNames(category *index.CategoryInfo) []string {
	var result []string

	if category == nil {
		result = append(result, NONE_VERSION)
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

	return result
}

// printCurrentVersionName print version from given slice for
// versions listing
func printCurrentVersionName(category string, names []string, installed map[string]bool, index int) bool {
	if len(names) > index {
		curName := names[index]

		if curName == NONE_VERSION {
			printSized(" {s}%%-%ds{!} ", categorySize[category], curName)
			return true
		}

		var prettyName string

		if strings.Contains(curName, "-railsexpress") {
			baseName := strings.Replace(curName, "-railsexpress", "", -1)

			switch {
			case installed[curName] && installed[baseName]:
				prettyName = fmt.Sprintf("%s{s}-railsexpress{!} {%s}••{!}", baseName, categoryColor[category])
			case installed[curName]:
				prettyName = fmt.Sprintf("%s{s}-railsexpress{!} {s}•{%s}•{!}", baseName, categoryColor[category])
			case installed[baseName]:
				prettyName = fmt.Sprintf("%s{s}-railsexpress{!} {%s}•{s}•{!}", baseName, categoryColor[category])
			default:
				prettyName = fmt.Sprintf("%s{s}-railsexpress{!}", baseName)
			}

			printRubyVersion(category, prettyName)

			return true
		}

		if installed[curName] {
			prettyName = fmt.Sprintf("%s {%s}•{!}", curName, categoryColor[category])
			printRubyVersion(category, prettyName)
		} else {
			printSized(" %%-%ds ", categorySize[category], curName)
		}

		return true
	}

	printSized(" %%-%ds ", categorySize[category], "")

	return false
}

// printSized render format with given size and print text with give arguments
func printSized(format string, size int, a ...interface{}) {
	fmtc.Printf(fmtc.Sprintf(format, size), a...)
}

// printRubyVersion print version with align spaces
func printRubyVersion(category, name string) {
	fmtc.Printf(" " + name + getAlignSpaces(fmtc.Clean(name), categorySize[category]) + " ")
}

// configureCategorySizes configure column size for each category
func configureCategorySizes(names map[string][]string) {
	terminalWidth, _ := terminal.GetSize()

	if terminalWidth == -1 || terminalWidth >= 140 {
		categorySize[CATEGORY_RUBY] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_JRUBY] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_REE] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_RUBINIUS] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_OTHER] = DEFAULT_CATEGORY_SIZE

		return
	}

	averageCategorySize := (terminalWidth - 10) / len(names)
	averageSize := terminalWidth - 10
	averageItems := 0

	for category, nameSlice := range names {
		for _, curName := range nameSlice {
			curNameLen := len(curName) + 4 // 4 for bullets

			if categorySize[category] < curNameLen {
				categorySize[category] = curNameLen
			}
		}

		if categorySize[category] > averageCategorySize {
			averageSize -= categorySize[category]
		} else {
			averageItems++
		}
	}

	if averageItems > 0 {
		for category, size := range categorySize {
			if size < averageCategorySize {
				categorySize[category] = averageSize / averageItems
			}
		}
	}
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

// isVersionInstalled return true is given version already installed
func isVersionInstalled(rubyVersion string) bool {
	fullPath := getVersionPath(rubyVersion)
	return fsutil.IsExist(fullPath)
}

// getInstalledVersionsMap return map with names of installed versions
func getInstalledVersionsMap() map[string]bool {
	result := make(map[string]bool)
	versions := fsutil.List(
		getRBEnvVersionsPath(), true,
		&fsutil.ListingFilter{Perms: "D"},
	)

	if len(versions) == 0 {
		return result
	}

	for _, version := range versions {
		result[version] = true
	}

	return result
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
	return getRBEnvVersionsPath() + "/" + rubyVersion
}

// getRBEnvVersionsPath return path to rbenv directory with all versions
func getRBEnvVersionsPath() string {
	return knf.GetS(RBENV_DIR) + "/versions"
}

// getAlignSpaces return spaces for message align
func getAlignSpaces(t string, l int) string {
	spaces := "                                    "
	return spaces[0 : l-utf8.RuneCount([]byte(t))]
}

// getGemSourceURL return url of gem source
func getGemSourceURL() string {
	if !arg.GetB(ARG_GEMS_INSECURE) && knf.GetB(GEMS_SOURCE_SECURE, false) {
		return "https://" + knf.GetS(GEMS_SOURCE)
	}

	return "http://" + knf.GetS(GEMS_SOURCE)
}

// checkRBEnvDirPerms check permissions on rbenv directory
func checkRBEnvDirPerms() {
	if !fsutil.CheckPerms("DWX", knf.GetS(RBENV_DIR)) {
		printError("Directory %s must be writable and executable", knf.GetS(RBENV_DIR))
		exit(1)
	}
}

// checkDependencies check dependencies for given category
func checkDependencies(category string) {
	if category != CATEGORY_JRUBY {
		return
	}

	if env.Which("java") == "" {
		printError("Can't find java binary on system. Java 1.6+ is required for all JRuby versions.")
		exit(1)
	}
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

	os.Chown(tmpName, currentUser.RealUID, currentUser.RealGID)

	return tmpName, nil
}

// intSignalHandler is INT (Ctrl+C) signal handler
func intSignalHandler() {
	printWarn("\n\nInstall process canceled by Ctrl+C")
	exit(1)
}

// printError prints error message to console
func printError(f string, a ...interface{}) {
	fmtc.Printf("{r}"+f+"{!}\n", a...)
}

// printError prints warning message to console
func printWarn(f string, a ...interface{}) {
	fmtc.Printf("{y}"+f+"{!}\n", a...)
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
	info.AddOption(ARG_RUBY_VERSION, "Install version defined in version file")
	info.AddOption(ARG_NO_COLOR, "Disable colors in output")
	info.AddOption(ARG_NO_PROGRESS, "Disable progress bar and spinner")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VER, "Show version")

	info.AddExample("2.0.0-p598", "Install 2.0.0-p598")
	info.AddExample("2.0.0-p598-railsexpress", "Install 2.0.0-p598 with railsexpress patches")
	info.AddExample("2.0.0-p598 -g", "Update gems installed on 2.0.0-p598")
	info.AddExample("-r", "Install version defined in .ruby-version file")

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

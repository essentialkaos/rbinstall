package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2018 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v9/env"
	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/fmtutil"
	"pkg.re/essentialkaos/ek.v9/fsutil"
	"pkg.re/essentialkaos/ek.v9/hash"
	"pkg.re/essentialkaos/ek.v9/knf"
	"pkg.re/essentialkaos/ek.v9/log"
	"pkg.re/essentialkaos/ek.v9/options"
	"pkg.re/essentialkaos/ek.v9/req"
	"pkg.re/essentialkaos/ek.v9/signal"
	"pkg.re/essentialkaos/ek.v9/sortutil"
	"pkg.re/essentialkaos/ek.v9/strutil"
	"pkg.re/essentialkaos/ek.v9/system"
	"pkg.re/essentialkaos/ek.v9/terminal"
	"pkg.re/essentialkaos/ek.v9/terminal/window"
	"pkg.re/essentialkaos/ek.v9/tmp"
	"pkg.re/essentialkaos/ek.v9/usage"
	"pkg.re/essentialkaos/ek.v9/usage/update"
	"pkg.re/essentialkaos/ek.v9/version"

	"pkg.re/essentialkaos/z7.v7"

	"pkg.re/cheggaaa/pb.v1"

	"github.com/essentialkaos/rbinstall/index"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "RBInstall"
	VER  = "0.19.0"
	DESC = "Utility for installing prebuilt ruby versions to rbenv"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// List of supported command-line arguments
const (
	OPT_REINSTALL     = "R:reinstall"
	OPT_UNINSTALL     = "U:uninstall"
	OPT_GEMS_UPDATE   = "G:gems-update"
	OPT_REHASH        = "H:rehash"
	OPT_GEMS_INSECURE = "s:gems-insecure"
	OPT_RUBY_VERSION  = "r:ruby-version"
	OPT_ALL           = "a:all"
	OPT_NO_COLOR      = "nc:no-color"
	OPT_NO_PROGRESS   = "np:no-progress"
	OPT_HELP          = "h:help"
	OPT_VER           = "v:version"
)

// List of supported config values
const (
	MAIN_TMP_DIR          = "main:tmp-dir"
	STORAGE_URL           = "storage:url"
	PROXY_ENABLED         = "proxy:enabled"
	PROXY_URL             = "proxy:url"
	RBENV_DIR             = "rbenv:dir"
	RBENV_ALLOW_OVERWRITE = "rbenv:allow-overwrite"
	RBENV_ALLOW_UNINSTALL = "rbenv:allow-uninstall"
	RBENV_MAKE_ALIAS      = "rbenv:make-alias"
	GEMS_RUBYGEMS_UPDATE  = "gems:rubygems-update"
	GEMS_RUBYGEMS_VERSION = "gems:rubygems-version"
	GEMS_ALLOW_UPDATE     = "gems:allow-update"
	GEMS_NO_DOCUMENT      = "gems:no-document"
	GEMS_SOURCE           = "gems:source"
	GEMS_SOURCE_SECURE    = "gems:source-secure"
	GEMS_INSTALL          = "gems:install"
	LOG_DIR               = "log:dir"
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

// Name of index file
const INDEX_NAME = "index.json"

// Path to config file
const CONFIG_FILE = "/etc/rbinstall.knf"

// Name of log with failed actions (gem install)
const FAIL_LOG_NAME = "rbinstall-fail.log"

// Value for column without any versions
const NONE_VERSION = "- none -"

// Default category column size
const DEFAULT_CATEGORY_SIZE = 28

// Default arch names
const (
	ARCH_X32 = "x32"
	ARCH_X64 = "x64"
	ARCH_ARM = "arm"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// PassThru is reader for progress bar
type PassThru struct {
	io.Reader
	pb *pb.ProgressBar
}

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_REINSTALL:     {Type: options.BOOL, Conflicts: OPT_UNINSTALL},
	OPT_UNINSTALL:     {Type: options.BOOL, Conflicts: OPT_REINSTALL},
	OPT_GEMS_UPDATE:   {Type: options.BOOL},
	OPT_GEMS_INSECURE: {Type: options.BOOL},
	OPT_RUBY_VERSION:  {Type: options.BOOL},
	OPT_REHASH:        {Type: options.BOOL},
	OPT_ALL:           {Type: options.BOOL},
	OPT_NO_COLOR:      {Type: options.BOOL},
	OPT_NO_PROGRESS:   {Type: options.BOOL},
	OPT_HELP:          {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:           {Type: options.BOOL, Alias: "ver"},
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

var categorySize = map[string]int{
	CATEGORY_RUBY:     0,
	CATEGORY_JRUBY:    0,
	CATEGORY_REE:      0,
	CATEGORY_RUBINIUS: 0,
	CATEGORY_OTHER:    0,
}

var useRawOutput = false

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	var err error
	var errs []error

	runtime.GOMAXPROCS(2)

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		fmtc.NewLine()

		for _, err = range errs {
			terminal.PrintErrorMessage(err.Error())
		}

		exit(1)
	}

	configureUI()

	if options.GetB(OPT_VER) {
		showAbout()
		return
	}

	if options.GetB(OPT_HELP) {
		showUsage()
		return
	}

	if !useRawOutput {
		fmtc.NewLine()
	}

	prepare()

	if options.GetB(OPT_REHASH) {
		rehashShims()
	} else {
		fetchIndex()
		process(args)
	}

	exit(0)
}

// configureUI configure user interface
func configureUI() {
	envVars := env.Get()
	term := envVars.GetS("TERM")

	fmtc.DisableColors = true
	fmtutil.SizeSeparator = " "

	if term != "" {
		switch {
		case strings.Contains(term, "xterm"),
			strings.Contains(term, "color"),
			term == "screen":
			fmtc.DisableColors = false
		}
	}

	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if !fsutil.IsCharacterDevice("/dev/stdout") && envVars.GetS("FAKETTY") == "" {
		fmtc.DisableColors = true
		useRawOutput = true
	}
}

// prepare do some preparations for installing ruby
func prepare() {
	req.SetUserAgent(APP, VER)

	loadConfig()
	validateConfig()
	configureProxy()
	setEnvVars()

	signal.Handlers{
		signal.INT: intSignalHandler,
	}.TrackAsync()
}

// configureProxy configure proxy settings
func configureProxy() {
	if !knf.GetB(PROXY_ENABLED, false) || !knf.HasProp(PROXY_URL) {
		return
	}

	proxyURL, err := url.Parse(knf.GetS(PROXY_URL))

	if err != nil {
		printErrorAndExit("Can't parse proxy URL: %v", err)
	}

	os.Setenv("http_proxy", knf.GetS(PROXY_URL))
	os.Setenv("https_proxy", knf.GetS(PROXY_URL))
	os.Setenv("HTTP_PROXY", knf.GetS(PROXY_URL))
	os.Setenv("HTTPS_PROXY", knf.GetS(PROXY_URL))

	req.Global.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
}

// setEnvVars set environment variables if rbenv is not initialized
func setEnvVars() {
	ev := env.Get()

	if ev.GetS("RBENV_ROOT") != "" {
		return
	}

	rbenvDir := knf.GetS(RBENV_DIR)
	newPath := rbenvDir + "/bin:"
	newPath += rbenvDir + "/libexec:"
	newPath += ev.GetS("PATH")

	os.Setenv("RBENV_ROOT", rbenvDir)
	os.Setenv("PATH", newPath)
}

// checkPerms check user for sudo
func checkPerms() {
	var err error

	currentUser, err = system.CurrentUser()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if !currentUser.IsRoot() {
		printErrorAndExit("This action requires superuser (root) privileges")
	}
}

// setupLogger setup logging subsystem
func setupLogger() {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS))

	if err != nil {
		printErrorAndExit(err.Error())
	}

	log.MinLevel(knf.GetI(LOG_LEVEL))
}

// setupTemp setup dir for temporary data
func setupTemp() {
	var err error

	temp, err = tmp.NewTemp(knf.GetS(MAIN_TMP_DIR, "/tmp"))

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// loadConfig load global config
func loadConfig() {
	err := knf.Global(CONFIG_FILE)

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// validateConfig validate knf.values
func validateConfig() {
	var permsChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !fsutil.CheckPerms(value.(string), config.GetS(prop)) {
			switch value.(string) {
			case "DWX":
				return fmtc.Errorf("Property %s must be path to writable directory", prop)
			}
		}

		return nil
	}

	errs := knf.Validate([]*knf.Validator{
		{MAIN_TMP_DIR, permsChecker, "DWX"},
		{STORAGE_URL, knf.Empty, nil},
	})

	if len(errs) != 0 {
		terminal.PrintErrorMessage("Error while config validation:")

		for _, err := range errs {
			terminal.PrintErrorMessage("  %v", err)
		}

		exit(1)
	}
}

// fetchIndex download index from remote repository
func fetchIndex() {
	resp, err := req.Request{URL: knf.GetS(STORAGE_URL) + "/" + INDEX_NAME}.Get()

	if err != nil {
		printErrorAndExit("Can't fetch repository index: %v", err)
	}

	if resp.StatusCode != 200 {
		printErrorAndExit("Can't fetch repository index: CDN return status code %d", resp.StatusCode)
	}

	repoIndex = index.NewIndex()

	err = resp.JSON(repoIndex)

	if err != nil {
		printErrorAndExit("Can't decode repository index json: %v", err)
	}

	repoIndex.Sort()
}

// process process command
func process(args []string) {
	var err error
	var rubyVersion string

	if len(args) != 0 {
		rubyVersion = args[0]
	} else if options.GetB(OPT_RUBY_VERSION) {
		rubyVersion, err = getVersionFromFile()

		if err != nil {
			printErrorAndExit(err.Error())
		}

		fmtc.Printf("{s}Installing version {s*}%s{s} from version file{!}\n\n", rubyVersion)
	}

	if rubyVersion != "" {
		checkPerms()
		setupLogger()
		setupTemp()

		if options.GetB(OPT_GEMS_UPDATE) {
			updateGems(rubyVersion)
		} else if options.GetB(OPT_UNINSTALL) {
			uninstallCommand(rubyVersion)
		} else {
			installCommand(rubyVersion)
		}
	} else {
		listCommand()
	}
}

// listCommand show list of all available versions
func listCommand() {
	dist, arch, err := getSystemInfo()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if !repoIndex.HasData(dist, arch) {
		terminal.PrintWarnMessage("Prebuilt binaries not found for this system")
		exit(0)
	}

	if useRawOutput {
		printRawListing(dist, arch)
	} else {
		printPrettyListing(dist, arch)
	}
}

// printPrettyListing print info about listing with colors in table view
func printPrettyListing(dist, arch string) {
	var (
		ruby     = getCategoryData(dist, arch, CATEGORY_RUBY)
		jruby    = getCategoryData(dist, arch, CATEGORY_JRUBY)
		ree      = getCategoryData(dist, arch, CATEGORY_REE)
		rubinius = getCategoryData(dist, arch, CATEGORY_RUBINIUS)
		other    = getCategoryData(dist, arch, CATEGORY_OTHER)

		installed = getInstalledVersionsMap()
	)

	configureCategorySizes(map[string]index.CategoryData{
		CATEGORY_RUBY:     ruby,
		CATEGORY_JRUBY:    jruby,
		CATEGORY_REE:      ree,
		CATEGORY_RUBINIUS: rubinius,
		CATEGORY_OTHER:    other,
	})

	headerTemplate := fmt.Sprintf(
		"{*@y} %%-%ds{!} {*@c} %%-%ds{!} {*@g} %%-%ds{!} {*@m} %%-%ds{!} {*@s} %%-%ds{!}\n\n",
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

	var index int

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

	if !options.GetB(OPT_ALL) {
		fmtc.NewLine()
		fmtc.Println("{s-}For listing outdated versions use option '--all'{!}")
	}
}

// printRawListing just print version names
func printRawListing(dist, arch string) {
	var result []string

	for _, category := range repoIndex.Data[dist][arch] {
		for _, version := range category {

			if version.EOL && !options.GetB(OPT_ALL) {
				continue
			}

			result = append(result, version.Name)

			if len(version.Variations) != 0 {
				for _, variation := range version.Variations {
					result = append(result, variation.Name)
				}
			}
		}
	}

	sortutil.Versions(result)

	fmt.Print(strings.Join(result, "\n"))
}

// getCategoryData return filtered versions slice
func getCategoryData(dist, arch, category string) index.CategoryData {
	if options.GetB(OPT_ALL) {
		return repoIndex.Data[dist][arch][category]
	}

	var result = index.CategoryData{}

	for _, version := range repoIndex.Data[dist][arch][category] {
		if version.EOL {
			continue
		}

		result = append(result, version)
	}

	return result
}

// installCommand install some version of ruby
func installCommand(rubyVersion string) {
	osName, archName, err := getSystemInfo()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	info, category := repoIndex.Find(osName, archName, rubyVersion)

	if info == nil {
		printErrorAndExit("Can't find info about version %s", rubyVersion)
	}

	checkRBEnv()
	checkDependencies(category)

	if isVersionInstalled(info.Name) {
		if knf.GetB(RBENV_ALLOW_OVERWRITE) && options.GetB(OPT_REINSTALL) {
			fmtc.Printf("{y}Reinstalling %s…{!}\n\n", info.Name)
		} else {
			terminal.PrintWarnMessage("Version %s already installed", info.Name)
			exit(0)
		}
	}

	if !fsutil.IsExist(getUnpackDirPath()) {
		err = os.Mkdir(getUnpackDirPath(), 0770)

		if err != nil {
			printErrorAndExit("Can't create directory for unpacking data: %v", err)
		}
	} else {
		os.Remove(getUnpackDirPath() + "/" + info.Name)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	fmtc.Printf("Fetching {c}%s {s-}(%s){!}…\n", info.Name, fmtutil.PrettySize(info.Size))

	url := knf.GetS(STORAGE_URL) + "/" + info.Path + "/" + info.File
	file, err := downloadFile(url, info.File)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	checkHashTask := &Task{
		Desc:    "Checking sha1 checksum",
		Handler: checkHashTaskHandler,
	}

	_, err = checkHashTask.Start(file, info.Hash)

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	unpackTask := &Task{
		Desc:    "Unpacking 7z archive",
		Handler: unpackTaskHandler,
	}

	_, err = unpackTask.Start(file, getUnpackDirPath())

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	checkBinaryTask := &Task{
		Desc:    "Checking binary",
		Handler: checkBinaryTaskHandler,
	}

	_, err = checkBinaryTask.Start(info.Name, getUnpackDirPath())

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if isVersionInstalled(info.Name) {
		if knf.GetB(RBENV_ALLOW_OVERWRITE) && options.GetB(OPT_REINSTALL) {
			err = os.RemoveAll(getVersionPath(info.Name))

			if err != nil {
				printErrorAndExit("Can't remove %s: %v", info.Name, err)
			}
		}
	}

	err = os.Rename(getUnpackDirPath()+"/"+info.Name, getVersionPath(info.Name))

	if err != nil {
		printErrorAndExit("Can't move unpacked data to rbenv directory: %v", err)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetB(GEMS_RUBYGEMS_UPDATE) {
		rgVersion := knf.GetS(GEMS_RUBYGEMS_VERSION, "latest")
		updRubygemsTask := &Task{
			Desc:    fmtc.Sprintf("Updating RubyGems to %s", rgVersion),
			Handler: updateRubygemsTaskHandler,
		}

		_, err = updRubygemsTask.Start(info.Name, rgVersion)

		if err != nil {
			terminal.PrintWarnMessage(err.Error())
		}
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
				fmtc.NewLine()
				printErrorAndExit(err.Error())
			}
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	var cleanVersionName string
	var aliasCreated bool

	if strings.Contains(info.Name, "-p0") {
		cleanVersionName = getNameWithoutPatchLevel(info.Name)

		if knf.GetB(RBENV_MAKE_ALIAS, false) && !fsutil.IsExist(getVersionPath(cleanVersionName)) {
			err = os.Symlink(getVersionPath(info.Name), getVersionPath(cleanVersionName))

			if err != nil {
				fmtc.Println("{r}✖ {!}Creating alias")
				terminal.PrintWarnMessage(err.Error())
			} else {
				fmtc.Println("{g}✔ {!}Creating alias")
				aliasCreated = true
			}
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	rehashShims()

	fmtc.NewLine()

	if aliasCreated {
		log.Info("[%s] Installed version %s as %s", currentUser.RealName, info.Name, cleanVersionName)
		fmtc.Printf("{g}Version {*}%s{!*} successfully installed as {*}%s{!}\n", info.Name, cleanVersionName)
	} else {
		log.Info("[%s] Installed version %s", currentUser.RealName, info.Name)
		fmtc.Printf("{g}Version {*}%s{!*} successfully installed{!}\n", info.Name)
	}
}

// uninstallCommand unistall some version of ruby
func uninstallCommand(rubyVersion string) {
	if !knf.GetB(RBENV_ALLOW_UNINSTALL, false) {
		printErrorAndExit("Uninstalling is not allowed")
	}

	osName, archName, err := getSystemInfo()

	if err != nil {
		printErrorAndExit("%v", err)
	}

	info, _ := repoIndex.Find(osName, archName, rubyVersion)

	if info == nil {
		printErrorAndExit("Can't find info about version %s", rubyVersion)
	}

	if !isVersionInstalled(info.Name) {
		printErrorAndExit("Version %s is not installed", rubyVersion)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	unistallTask := &Task{
		Desc:    fmt.Sprintf("Unistalling %s", rubyVersion),
		Handler: unistallTaskHandler,
	}

	_, err = unistallTask.Start(info.Name)

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	rehashShims()

	fmtc.NewLine()

	log.Info("[%s] Uninstalled version %s", currentUser.RealName, info.Name)
	fmtc.Printf("{g}Version {*}%s{!*} successfully uninstalled{!}\n", rubyVersion)
}

// rehashShims run 'rbenv rehash' command
func rehashShims() {
	rehashTask := &Task{
		Desc:    "Rehashing",
		Handler: rehashTaskHandler,
	}

	_, err := rehashTask.Start()

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}
}

// unistallTaskHandler remove data for given ruby version
func unistallTaskHandler(args ...string) (string, error) {
	versionName := args[0]

	versionsDir := getRBEnvVersionsPath()
	cleanVersionName := getNameWithoutPatchLevel(versionName)

	var err error

	// Remove symlink
	if fsutil.IsExist(versionsDir + "/" + cleanVersionName) {
		err = os.Remove(versionsDir + "/" + cleanVersionName)

		if err != nil {
			return "", err
		}
	}

	// Remove directory with files
	if fsutil.IsExist(versionsDir + "/" + versionName) {
		err = os.RemoveAll(versionsDir + "/" + versionName)

		if err != nil {
			return "", err
		}
	}

	return "", nil
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

// checkHashTaskHandler check archive checksum
func checkHashTaskHandler(args ...string) (string, error) {
	filePath := args[0]
	fileHash := args[1]

	curHash := hash.FileHash(filePath)

	if fileHash != curHash {
		return "", fmtc.Errorf("Wrong file hash %s ≠ %s", fileHash, curHash)
	}

	return "", nil
}

// unpackTaskHandler run unpacking command
func unpackTaskHandler(args ...string) (string, error) {
	file := args[0]
	outputDir := args[1]

	output, err := z7.Extract(z7.Props{File: file, OutputDir: outputDir})

	if err != nil {
		unpackError := err
		actionLog, err := logFailedAction(output)

		if err != nil {
			return "", fmtc.Errorf("7za return error: %s", unpackError.Error())
		}

		return "", fmtc.Errorf("7za return error: %s (7za output saved as %s)", unpackError.Error(), actionLog)
	}

	return "", nil
}

// checkBinaryTaskHandler run and check installer binary
func checkBinaryTaskHandler(args ...string) (string, error) {
	version := args[0]
	unpackDir := args[1]

	binary := unpackDir + "/" + version + "/bin/ruby"

	err := exec.Command(binary, "--version").Start()

	return "", err
}

// installGemTaskHandler run gems installing command
func installGemTaskHandler(args ...string) (string, error) {
	version := args[0]
	gem := args[1]

	return runGemCmd(version, "install", gem)
}

// updateGemTaskHandler run gems update command
func updateGemTaskHandler(args ...string) (string, error) {
	version := args[0]
	gem := args[1]

	return runGemCmd(version, "update", gem)
}

// updateRubygemsTaskHandler run rubygems update command
func updateRubygemsTaskHandler(args ...string) (string, error) {
	version := args[0]
	rgVersion := args[1]

	return "", updateRubygems(version, rgVersion)
}

// rehashTaskHandler run 'rbenv rehash' command
func rehashTaskHandler(args ...string) (string, error) {
	rehashCmd := exec.Command("rbenv", "rehash")
	output, err := rehashCmd.CombinedOutput()

	if err != nil {
		return "", errors.New(strings.TrimRight(string(output), "\r\n"))
	}

	return "", nil
}

// updateGems update gems installed by rbinstall on defined version
func updateGems(rubyVersion string) {
	if !knf.GetB(GEMS_ALLOW_UPDATE, true) {
		printErrorAndExit("Gems update is disabled in configuration file")
	}

	fullPath := getVersionPath(rubyVersion)

	if !fsutil.IsExist(fullPath) {
		printErrorAndExit("Version %s is not installed", rubyVersion)
	}

	checkRBEnv()

	runDate = time.Now()
	installed := false

	fmtc.Printf("Updating gems for {c}%s{!}…\n\n", rubyVersion)

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetB(GEMS_RUBYGEMS_UPDATE) {
		rgVersion := knf.GetS(GEMS_RUBYGEMS_VERSION, "latest")
		updRubygemsTask := &Task{
			Desc:    fmtc.Sprintf("Updating RubyGems to %s", rgVersion),
			Handler: updateRubygemsTaskHandler,
		}

		updRubygemsTask.Start(rubyVersion, rgVersion)

		installed = true
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetS(GEMS_INSTALL) != "" {
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
				fmtc.NewLine()
				printErrorAndExit(err.Error())
			}
		}

		installed = true
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if installed {
		rehashShims()

		fmtc.NewLine()
		fmtc.Println("{g}All gems successfully updated!{!}")
	} else {
		fmtc.NewLine()
		fmtc.Println("{y}There is nothing to update{!}")
	}
}

// runGemCmd run some gem command for some version
func runGemCmd(rubyVersion, cmd, gem string) (string, error) {
	start := time.Now()
	rubyPath := getVersionPath(rubyVersion)
	gemCmd := exec.Command(rubyPath+"/bin/ruby", rubyPath+"/bin/gem", cmd, gem)

	if knf.GetB(GEMS_NO_DOCUMENT) {
		gemCmd.Args = append(gemCmd.Args, "--no-document")
	}

	if knf.GetS(GEMS_SOURCE) != "" {
		gemCmd.Args = append(gemCmd.Args, "--source", getGemSourceURL())
	}

	output, err := gemCmd.CombinedOutput()

	if err == nil {
		version := getInstalledGemVersion(rubyVersion, gem, start)

		if version == "" {
			return "", nil
		}

		return version, nil
	}

	actionLog, err := logFailedAction(strings.TrimRight(string(output), "\r\n"))

	if err == nil {
		switch cmd {
		case "update":
			return "", fmtc.Errorf("Can't update gem %s. Gem command output saved as %s.", gem, actionLog)
		default:
			return "", fmtc.Errorf("Can't install gem %s. Gem command output saved as %s.", gem, actionLog)
		}
	}

	switch cmd {
	case "update":
		return "", fmtc.Errorf("Can't update gem %s", gem)
	default:
		return "", fmtc.Errorf("Can't install gem %s", gem)
	}
}

// updateRubygems update rubygems to defined version
func updateRubygems(version, rgVersion string) error {
	var gemCmd *exec.Cmd

	rubyPath := getVersionPath(version)

	if rgVersion == "latest" {
		gemCmd = exec.Command(
			rubyPath+"/bin/ruby", rubyPath+"/bin/gem",
			"update", "--system",
			"--source", getGemSourceURL(),
		)
	} else {
		gemCmd = exec.Command(
			rubyPath+"/bin/ruby", rubyPath+"/bin/gem",
			"update", "--system", rgVersion,
			"--source", getGemSourceURL(),
		)
	}

	output, err := gemCmd.CombinedOutput()

	if err == nil {
		return nil
	}

	actionLog, err := logFailedAction(strings.TrimRight(string(output), "\r\n"))

	if err == nil {
		return fmt.Errorf("Can't update rubygems. Update command output saved as %s", actionLog)
	}

	return fmt.Errorf("Can't update rubygems")
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

	if options.GetB(OPT_NO_PROGRESS) {
		_, err = io.Copy(fd, resp.Body)
	} else {
		bar := makeProgressBar(resp.ContentLength)

		defer bar.Finish()

		pt := &PassThru{
			Reader: resp.Body,
			pb:     bar.Start(),
		}

		_, err = io.Copy(fd, pt)
	}

	return tmpDir + "/" + fileName, err
}

// printCurrentVersionName print version from given slice for
// versions listing
func printCurrentVersionName(category string, versions index.CategoryData, installed map[string]bool, index int) bool {
	if len(versions) > index {
		curName := versions[index].Name

		var prettyName string

		if len(versions[index].Variations) != 0 {

			// Currently subversion is only one - railsexpress
			subVerName := versions[index].Variations[0].Name

			if options.GetB(OPT_NO_COLOR) {
				switch {
				case installed[curName] && installed[subVerName]:
					prettyName = fmt.Sprintf("%s-railsexpress ••", curName)
				case installed[subVerName]:
					prettyName = fmt.Sprintf("%s-railsexpress -•", curName)
				case installed[curName]:
					prettyName = fmt.Sprintf("%s-railsexpress •-", curName)
				default:
					prettyName = fmt.Sprintf("%s-railsexpress", curName)
				}
			} else {
				switch {
				case installed[curName] && installed[subVerName]:
					prettyName = fmt.Sprintf("%s{s-}-railsexpress{!} {%s}••{!}", curName, categoryColor[category])
				case installed[subVerName]:
					prettyName = fmt.Sprintf("%s{s-}-railsexpress{!} {s-}•{%s}•{!}", curName, categoryColor[category])
				case installed[curName]:
					prettyName = fmt.Sprintf("%s{s-}-railsexpress{!} {%s}•{s-}•{!}", curName, categoryColor[category])
				default:
					prettyName = fmt.Sprintf("%s{s-}-railsexpress{!}", curName)
				}
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

	if len(versions) == 0 && index == 0 {
		printSized(" {s-}%%-%ds{!} ", categorySize[category], NONE_VERSION)
		return true
	}

	printSized(" %%-%ds ", categorySize[category], "")

	return false
}

// makeProgressBar create and configure progress bar instance
func makeProgressBar(total int64) *pb.ProgressBar {
	bar := pb.New64(total)

	bar.ShowCounters = false
	bar.BarStart = "—"
	bar.BarEnd = " "
	bar.Empty = " "
	bar.Current = "—"
	bar.CurrentN = "→"
	bar.Width = 80
	bar.ForceWidth = false
	bar.RefreshRate = 50 * time.Millisecond

	return bar
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
func configureCategorySizes(data map[string]index.CategoryData) {
	terminalWidth := window.GetWidth()

	if terminalWidth == -1 || terminalWidth > 150 {
		categorySize[CATEGORY_RUBY] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_JRUBY] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_REE] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_RUBINIUS] = DEFAULT_CATEGORY_SIZE
		categorySize[CATEGORY_OTHER] = DEFAULT_CATEGORY_SIZE

		return
	}

	averageCategorySize := (terminalWidth - 10) / len(data)
	averageSize := terminalWidth - 10
	averageItems := 0

	for categoryName, categoryData := range data {
		for _, item := range categoryData {
			nameLen := len(item.Name) + 4 // 4 for bullets

			if categorySize[categoryName] < nameLen {
				categorySize[categoryName] = nameLen
			}

			if len(item.Variations) != 0 {
				for _, subVer := range item.Variations {
					nameLen = len(subVer.Name) + 4 // 4 for bullets

					if categorySize[categoryName] < nameLen {
						categorySize[categoryName] = nameLen
					}
				}
			}
		}

		if categorySize[categoryName] > averageCategorySize {
			averageSize -= categorySize[categoryName]
		} else {
			averageItems++
		}
	}

	if averageItems > 0 {
		for categoryName, size := range categorySize {
			if size < averageCategorySize {
				categorySize[categoryName] = averageSize / averageItems
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
		fsutil.ListingFilter{Perms: "D"},
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

// getUnpackDirPath return path to directory for unpacking data
func getUnpackDirPath() string {
	return getRBEnvVersionsPath() + "/.rbinstall"
}

// getAlignSpaces return spaces for output align
func getAlignSpaces(t string, l int) string {
	return strings.Repeat(" ", 36)[:l-strutil.Len(t)]
}

// getGemSourceURL return url of gem source
func getGemSourceURL() string {
	if !options.GetB(OPT_GEMS_INSECURE) && knf.GetB(GEMS_SOURCE_SECURE, false) {
		return "https://" + knf.GetS(GEMS_SOURCE)
	}

	return "http://" + knf.GetS(GEMS_SOURCE)
}

// checkRBEnv check RBEnv directory and state
func checkRBEnv() {
	versionsDir := getRBEnvVersionsPath()

	if !fsutil.CheckPerms("DWX", versionsDir) {
		printErrorAndExit("Directory %s must be writable and executable", versionsDir)
	}

	binary := knf.GetS(RBENV_DIR) + "/libexec/rbenv"

	if !fsutil.CheckPerms("FRX", binary) {
		printErrorAndExit("rbenv is not installed. Follow these instructions to install rbenv https://github.com/rbenv/rbenv#installation")
	}
}

// checkDependencies check dependencies for given category
func checkDependencies(category string) {
	if category != CATEGORY_JRUBY {
		return
	}

	if env.Which("java") == "" {
		printErrorAndExit("Can't find java binary on system. Java 1.6+ is required for all JRuby versions.")
	}
}

// getSystemInfo return info about system
func getSystemInfo() (string, string, error) {
	var (
		os   string
		arch string
	)

	systemInfo, err := system.GetSystemInfo()

	// Return by default x64
	if err != nil {
		return "", "", fmt.Errorf("Can't get information about system")
	}

	switch systemInfo.Arch {
	case "i386", "i586", "i686":
		arch = ARCH_X32
	case "x86_64":
		arch = ARCH_X64
	case "arm":
		arch = ARCH_ARM
	default:
		return "", "", fmt.Errorf("Architecture %s is not supported yet", systemInfo.Arch)
	}

	if strings.ToLower(systemInfo.OS) != "linux" {
		return "", "", fmt.Errorf("%s is not supported yet", systemInfo.OS)
	}

	distVersion, err := version.Parse(systemInfo.Version)

	if err != nil {
		return "", "", fmt.Errorf("Can't parse OS version")
	}

	os = fmt.Sprintf("%s-%d", strings.ToLower(systemInfo.Distribution), distVersion.Major())

	return os, arch, nil
}

// getNameWithoutPatchLevel return name without -p0
func getNameWithoutPatchLevel(name string) string {
	return strings.Replace(name, "-p0", "", -1)
}

// logFailedAction save data to temporary log file and return path
// to this log file
func logFailedAction(message string) (string, error) {
	if len(message) == 0 {
		return "", errors.New("Output data is empty")
	}

	tmpName := knf.GetS(MAIN_TMP_DIR) + "/" + FAIL_LOG_NAME

	if fsutil.IsExist(tmpName) {
		os.Remove(tmpName)
	}

	data := append([]byte(message), []byte("\n\n")...)

	err := ioutil.WriteFile(tmpName, data, 0666)

	if err != nil {
		return "", err
	}

	os.Chown(tmpName, currentUser.RealUID, currentUser.RealGID)

	return tmpName, nil
}

// intSignalHandler is INT (Ctrl+C) signal handler
func intSignalHandler() {
	printErrorAndExit("\n\nInstall process canceled by Ctrl+C")
}

// printErrorAndExit print error message and exit with non-zero exit code
func printErrorAndExit(f string, a ...interface{}) {
	terminal.PrintErrorMessage(f, a...)
	exit(1)
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

	info.AddOption(OPT_REINSTALL, "Reinstall already installed version {s-}(if allowed in config){!}")
	info.AddOption(OPT_UNINSTALL, "Uninstall already installed version {s-}(if allowed in config){!}")
	info.AddOption(OPT_GEMS_UPDATE, "Update gems for some version {s-}(if allowed in config){!}")
	info.AddOption(OPT_REHASH, "Rehash rbenv shims")
	info.AddOption(OPT_GEMS_INSECURE, "Use HTTP instead of HTTPS for installing gems")
	info.AddOption(OPT_RUBY_VERSION, "Install version defined in version file")
	info.AddOption(OPT_ALL, "Print all available versions")
	info.AddOption(OPT_NO_PROGRESS, "Disable progress bar and spinner")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample("2.0.0-p598", "Install 2.0.0-p598")
	info.AddExample("2.0.0", "Install latest available release in 2.0.0")
	info.AddExample("2.0.0-p598-railsexpress", "Install 2.0.0-p598 with railsexpress patches")
	info.AddExample("2.0.0-p598 -G", "Update gems installed for 2.0.0-p598")
	info.AddExample("2.0.0-p598 --reinstall", "Reinstall 2.0.0-p598")
	info.AddExample("-r", "Install version defined in .ruby-version file")

	info.Render()
}

func showAbout() {
	about := &usage.About{
		App:           APP,
		Version:       VER,
		Desc:          DESC,
		Year:          2006,
		Owner:         "ESSENTIAL KAOS",
		License:       "Essential Kaos Open Source License <https://essentialkaos.com/ekol>",
		UpdateChecker: usage.UpdateChecker{"essentialkaos/rbinstall", update.GitHubChecker},
	}

	about.Render()
}

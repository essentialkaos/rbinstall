package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2022 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
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
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/env"
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/hash"
	"github.com/essentialkaos/ek/v12/knf"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/passwd"
	"github.com/essentialkaos/ek/v12/progress"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/signal"
	"github.com/essentialkaos/ek/v12/sortutil"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/terminal/window"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/tmp"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"
	"github.com/essentialkaos/ek/v12/usage/update"
	"github.com/essentialkaos/ek/v12/version"

	knfv "github.com/essentialkaos/ek/v12/knf/validators"
	knff "github.com/essentialkaos/ek/v12/knf/validators/fs"

	"github.com/essentialkaos/zip7"

	"github.com/essentialkaos/rbinstall/index"
	"github.com/essentialkaos/rbinstall/support"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "RBInstall"
	VER  = "2.4.0"
	DESC = "Utility for installing prebuilt Ruby versions to RBEnv"
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
	OPT_INFO          = "i:info"
	OPT_ALL           = "a:all"
	OPT_NO_COLOR      = "nc:no-color"
	OPT_NO_PROGRESS   = "np:no-progress"
	OPT_HELP          = "h:help"
	OPT_VER           = "v:version"

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
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

// INDEX_NAME is name of index file
const INDEX_NAME = "index.json"

// CONFIG_FILE is path to config file
const CONFIG_FILE = "/etc/rbinstall.knf"

// NONE_VERSION is value for column without any versions
const NONE_VERSION = "- none -"

// DEFAULT_CATEGORY_SIZE is default category column size
const DEFAULT_CATEGORY_SIZE = 22

// Default arch names
const (
	ARCH_X32 = "x32"
	ARCH_X64 = "x64"
	ARCH_ARM = "arm"
)

// RubyGems versions used for old versions of Ruby
const (
	MIN_RUBYGEMS_VERSION_BASE  = "2.7.9"
	MIN_RUBYGEMS_VERSION_JRUBY = "2.6.14"
)

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_REINSTALL:     {Type: options.BOOL, Conflicts: OPT_UNINSTALL},
	OPT_UNINSTALL:     {Type: options.BOOL, Conflicts: OPT_REINSTALL},
	OPT_GEMS_UPDATE:   {Type: options.BOOL},
	OPT_GEMS_INSECURE: {Type: options.BOOL},
	OPT_RUBY_VERSION:  {Type: options.BOOL},
	OPT_REHASH:        {Type: options.BOOL},
	OPT_ALL:           {Type: options.BOOL},
	OPT_INFO:          {Type: options.BOOL},
	OPT_NO_COLOR:      {Type: options.BOOL},
	OPT_NO_PROGRESS:   {Type: options.BOOL},
	OPT_HELP:          {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:           {Type: options.BOOL, Alias: "ver"},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

var repoIndex *index.Index
var temp *tmp.Temp
var currentUser *system.User
var runDate time.Time

var categoryColor = map[string]string{
	index.CATEGORY_RUBY:    "m",
	index.CATEGORY_JRUBY:   "c",
	index.CATEGORY_TRUFFLE: "y",
	index.CATEGORY_OTHER:   "s",
}

var categorySize = map[string]int{
	index.CATEGORY_RUBY:    0,
	index.CATEGORY_JRUBY:   0,
	index.CATEGORY_TRUFFLE: 0,
	index.CATEGORY_OTHER:   0,
}

var colorTagApp string
var colorTagVer string

var useRawOutput = false

// ////////////////////////////////////////////////////////////////////////////////// //

func Init(gitRev string, gomod []byte) {
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

	switch {
	case options.Has(OPT_COMPLETION):
		exit(genCompletion())
	case options.Has(OPT_GENERATE_MAN):
		exit(genMan())
	case options.GetB(OPT_VER):
		showAbout(gitRev)
		return
	case options.GetB(OPT_VERB_VER):
		showVerboseAbout(gitRev, gomod)
		return
	case options.GetB(OPT_HELP):
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

	if options.GetB(OPT_NO_PROGRESS) {
		spinner.DisableAnimation = true
	}

	if !fsutil.IsCharacterDevice("/dev/stdout") && envVars.GetS("FAKETTY") == "" {
		fmtc.DisableColors = true
		useRawOutput = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{#CC1E2C}", "{#CC1E2C}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{#160}", "{#160}"
	default:
		colorTagApp, colorTagVer = "{r}", "{r}"
	}

	progress.DefaultSettings.NameColorTag = "{*}"
	progress.DefaultSettings.PercentColorTag = "{*}"
	progress.DefaultSettings.ProgressColorTag = "{s}"
	progress.DefaultSettings.SpeedColorTag = "{s}"
	progress.DefaultSettings.RemainingColorTag = "{s}"
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
	errs := knf.Validate([]*knf.Validator{
		{MAIN_TMP_DIR, knff.Perms, "DWX"},
		{STORAGE_URL, knfv.Empty, nil},
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
		printErrorAndExit("Can't fetch repository index: storage return status code %d", resp.StatusCode)
	}

	repoIndex = index.NewIndex()

	err = resp.JSON(repoIndex)

	if err != nil {
		printErrorAndExit("Can't decode repository index json: %v", err)
	}

	repoIndex.Sort()
}

// process process command
func process(args options.Arguments) {
	var err error
	var rubyVersion string

	if len(args) != 0 {
		rubyVersion = args.Get(0).String()
	} else if options.GetB(OPT_RUBY_VERSION) {
		rubyVersion, err = getVersionFromFile()

		if err != nil {
			printErrorAndExit(err.Error())
		}

		fmtc.Printf("{s}Installing version {s*}%s{s} from version file{!}\n\n", rubyVersion)
	}

	if rubyVersion != "" {
		if options.GetB(OPT_INFO) {
			showDetailedInfo(rubyVersion)
			return
		}

		checkPerms()
		setupLogger()
		setupTemp()

		switch {
		case options.GetB(OPT_GEMS_UPDATE):
			updateGems(rubyVersion)
		case options.GetB(OPT_UNINSTALL):
			uninstallCommand(rubyVersion)
		default:
			installCommand(rubyVersion)
		}
	} else {
		listCommand()
	}
}

// showDetailedInfo shows detailed information about given version
func showDetailedInfo(rubyVersion string) {
	info, _, err := getVersionInfo(rubyVersion)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	fmtutil.Separator(true)

	url := fmt.Sprintf("%s/%s/%s", knf.GetS(STORAGE_URL), info.Path, info.File)
	added := timeutil.Format(time.Unix(info.Added, 0), "%Y/%m/%d %H:%M")

	fmtc.Printf(" {*}%-16s{!} {s}|{!} %s\n", "Name", info.Name)
	fmtc.Printf(" {*}%-16s{!} {s}|{!} %s\n", "URL", url)
	fmtc.Printf(" {*}%-16s{!} {s}|{!} %s\n", "Size", fmtutil.PrettySize(info.Size))
	fmtc.Printf(" {*}%-16s{!} {s}|{!} %s\n", "SHA-256 Checksum", info.Hash)
	fmtc.Printf(" {*}%-16s{!} {s}|{!} %s\n", "Added", added)

	if isVersionInstalled(info.Name) {
		installDate, _ := fsutil.GetMTime(getVersionPath(info.Name))
		installDateStr := timeutil.Format(installDate, "%Y/%m/%d %H:%M")
		fmtc.Printf(" {*}%-16s{!} {s}|{!} Yes {s-}(%s){!}\n", "Installed", installDateStr)
	} else {
		fmtc.Printf(" {*}%-16s{!} {s}|{!} No\n", "Installed")
	}

	if info.EOL {
		fmtc.Printf(" {*}%-16s{!} {s}|{!} {r}Yes{!}\n", "EOL")
	} else {
		fmtc.Printf(" {*}%-16s{!} {s}|{!} No\n", "EOL")
	}

	if len(info.Variations) != 0 {
		for index, variation := range info.Variations {
			if index == 0 {
				fmtc.Printf(
					" {*}%-16s{!} {s}|{!} %s {s-}(%s){!}\n",
					"Variations", variation.Name, fmtutil.PrettySize(variation.Size),
				)
			} else {
				fmtc.Printf(
					" {*}%-16s{!} {s}|{!} %s {s-}(%s){!}\n",
					"", variation.Name, fmtutil.PrettySize(variation.Size),
				)
			}
		}
	}

	fmtutil.Separator(true)
}

// listCommand show list of all available versions
func listCommand() {
	dist, arch, err := getSystemInfo()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if !repoIndex.HasData(dist, arch) {
		terminal.PrintWarnMessage(
			"Prebuilt binaries not found for this system (%s/%s)",
			dist, arch,
		)
		exit(1)
	}

	if useRawOutput {
		printRawListing(dist, arch)
	} else {
		printPrettyListing(dist, arch)
	}
}

// printPrettyListing print info about listing with colors in table view
func printPrettyListing(dist, arch string) {
	ruby := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_RUBY, options.GetB(OPT_ALL))
	jruby := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_JRUBY, options.GetB(OPT_ALL))
	truffle := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_TRUFFLE, options.GetB(OPT_ALL))
	other := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_OTHER, options.GetB(OPT_ALL))

	installed := getInstalledVersionsMap()

	configureCategorySizes(map[string]index.CategoryData{
		index.CATEGORY_RUBY:    ruby,
		index.CATEGORY_JRUBY:   jruby,
		index.CATEGORY_TRUFFLE: truffle,
		index.CATEGORY_OTHER:   other,
	})

	rubyTotal := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_RUBY, true).Total()
	jrubyTotal := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_JRUBY, true).Total()
	truffleTotal := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_TRUFFLE, true).Total()
	otherTotal := repoIndex.GetCategoryData(dist, arch, index.CATEGORY_OTHER, true).Total()

	headerTemplate := getCategoryHeaderStyle(index.CATEGORY_RUBY) + " " +
		getCategoryHeaderStyle(index.CATEGORY_JRUBY) + " " +
		getCategoryHeaderStyle(index.CATEGORY_TRUFFLE) + " " +
		getCategoryHeaderStyle(index.CATEGORY_OTHER) + "\n\n"

	fmtc.Printf(
		headerTemplate,
		fmt.Sprintf("%s (%d)", strings.ToUpper(index.CATEGORY_RUBY), rubyTotal),
		fmt.Sprintf("%s (%d)", strings.ToUpper(index.CATEGORY_JRUBY), jrubyTotal),
		fmt.Sprintf("%s (%d)", strings.ToUpper(index.CATEGORY_TRUFFLE), truffleTotal),
		fmt.Sprintf("%s (%d)", strings.ToUpper(index.CATEGORY_OTHER), otherTotal),
	)

	var counter int

	for {
		hasItems := false

		hasItems = printCurrentVersionName(index.CATEGORY_RUBY, ruby, installed, counter) || hasItems
		hasItems = printCurrentVersionName(index.CATEGORY_JRUBY, jruby, installed, counter) || hasItems
		hasItems = printCurrentVersionName(index.CATEGORY_TRUFFLE, truffle, installed, counter) || hasItems
		hasItems = printCurrentVersionName(index.CATEGORY_OTHER, other, installed, counter) || hasItems

		if !hasItems {
			break
		}

		fmtc.NewLine()

		counter++
	}

	if !options.GetB(OPT_ALL) {
		fmtc.NewLine()
		fmtc.Println("{s-}For listing outdated versions use option '--all'{!}")
	}
}

// getCategoryHeaderStyle generates part of the header style for given category
func getCategoryHeaderStyle(category string) string {
	return fmt.Sprintf(
		"{*@%s} %%-%ds{!}",
		categoryColor[category],
		categorySize[category],
	)
}

// printRawListing just print version names
func printRawListing(dist, arch string) {
	var result []string

	installed := getInstalledVersionsMap()

	for _, category := range repoIndex.Data[dist][arch] {
		for _, version := range category {

			if !installed[version.Name] && !options.GetB(OPT_ALL) {
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

	if len(result) == 0 {
		return
	}

	sortutil.Versions(result)

	fmt.Print(strings.Join(result, "\n"))
}

// installCommand install some version of ruby
func installCommand(rubyVersion string) {
	info, category, err := getVersionInfo(rubyVersion)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	checkRBEnv()
	checkDependencies(info, category)

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

	progress.DefaultSettings.BarFgColorTag = "{" + categoryColor[category] + "}"
	spinner.SpinnerColorTag = "{" + categoryColor[category] + "}"

	fmtc.Printf("Fetching {*}{"+categoryColor[category]+"}%s{!} from storage…\n", info.Name)

	file, err := downloadFile(info)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	spinner.Show("Checking SHA-1 checksum")
	err = checkHashTaskHandler(file, info.Hash)
	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	spinner.Show("Unpacking 7z archive")
	err = unpackTaskHandler(file, getUnpackDirPath())
	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	spinner.Show("Checking binary")
	err = checkBinaryTaskHandler(info.Name, getUnpackDirPath())
	spinner.Done(err == nil)

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
		rgVersion := getAdvisableRubyGemsVersion(info.Name)

		spinner.Show("Updating RubyGems to %s", rgVersion)
		err = updateRubygemsTaskHandler(info.Name, rgVersion)
		spinner.Done(err == nil)

		if err != nil {
			terminal.PrintWarnMessage(err.Error())
		}
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetS(GEMS_INSTALL) != "" {
		for _, gem := range strings.Split(knf.GetS(GEMS_INSTALL), " ") {
			gemName, gemVersion := parseGemInfo(gem)
			taskDesc := fmt.Sprintf("Installing %s", gemName)

			if gemVersion != "" {
				taskDesc += fmt.Sprintf(" (%s.x)", gemVersion)
			} else {
				taskDesc += fmt.Sprintf(" (latest)")
			}

			spinner.Show(taskDesc)
			_, err = installGemTaskHandler(info.Name, gemName, gemVersion)
			spinner.Done(err == nil)

			if err != nil {
				terminal.PrintWarnMessage(err.Error())
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
				fmtc.Println("{r}✖  {!}Creating alias")
				terminal.PrintWarnMessage(err.Error())
			} else {
				fmtc.Println("{g}✔  {!}Creating alias")
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

	info, _, err := getVersionInfo(rubyVersion)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if !isVersionInstalled(info.Name) {
		printErrorAndExit("Version %s is not installed", rubyVersion)
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	spinner.Show("Unistalling %s", rubyVersion)
	err = unistallTaskHandler(info.Name)
	spinner.Done(err == nil)

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
	spinner.Show("Rehashing")
	err := rehashTaskHandler()
	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		printErrorAndExit(err.Error())
	}
}

// unistallTaskHandler remove data for given ruby version
func unistallTaskHandler(versionName string) error {
	versionsDir := getRBEnvVersionsPath()
	cleanVersionName := getNameWithoutPatchLevel(versionName)

	var err error

	// Remove symlink
	if fsutil.IsExist(versionsDir + "/" + cleanVersionName) {
		err = os.Remove(versionsDir + "/" + cleanVersionName)

		if err != nil {
			return err
		}
	}

	// Remove directory with files
	if fsutil.IsExist(versionsDir + "/" + versionName) {
		err = os.RemoveAll(versionsDir + "/" + versionName)

		if err != nil {
			return err
		}
	}

	return nil
}

// checkHashTaskHandler check archive checksum
func checkHashTaskHandler(filePath, fileHash string) error {
	curHash := hash.FileHash(filePath)

	if fileHash != curHash {
		return fmt.Errorf("Wrong file hash %s ≠ %s", fileHash, curHash)
	}

	return nil
}

// unpackTaskHandler run unpacking command
func unpackTaskHandler(file, outputDir string) error {
	output, err := zip7.Extract(zip7.Props{File: file, OutputDir: outputDir})

	if err != nil {
		unpackError := err
		actionLog, err := logFailedAction(output)

		if err != nil {
			return fmtc.Errorf("7za return error: %s", unpackError.Error())
		}

		return fmtc.Errorf("7za return error: %s (7za output saved as %s)", unpackError.Error(), actionLog)
	}

	return nil
}

// checkBinaryTaskHandler run and check installer binary
func checkBinaryTaskHandler(args ...string) error {
	version, unpackDir := args[0], args[1]

	binary := unpackDir + "/" + version + "/bin/ruby"

	return exec.Command(binary, "--version").Start()
}

// installGemTaskHandler run gems installing command
func installGemTaskHandler(rubyVersion, gem, gemVersion string) (string, error) {
	// Do not install the latest version of bundler on Ruby < 2.3.0
	if gem == "bundler" && gemVersion == "" && !isVersionSupportedByBundler(rubyVersion) {
		return "", nil
	}

	return runGemCmd(rubyVersion, "install", gem, gemVersion)
}

// updateGemTaskHandler run gems update command
func updateGemTaskHandler(rubyVersion, gem, gemVersion string) (string, error) {
	// Do not install the latest version of bundler on Ruby < 2.3.0
	if gem == "bundler" && gemVersion == "" && !isVersionSupportedByBundler(rubyVersion) {
		return "", nil
	}

	if gemVersion != "" {
		return runGemCmd(rubyVersion, "install", gem, gemVersion)
	}

	return runGemCmd(rubyVersion, "update", gem, gemVersion)
}

// updateRubygemsTaskHandler run rubygems update command
func updateRubygemsTaskHandler(version, rgVersion string) error {
	return updateRubygems(version, rgVersion)
}

// rehashTaskHandler run 'rbenv rehash' command
func rehashTaskHandler() error {
	rehashCmd := exec.Command("rbenv", "rehash")
	output, err := rehashCmd.CombinedOutput()

	if err != nil {
		return errors.New(strings.TrimRight(string(output), "\r\n"))
	}

	return nil
}

// updateGems update gems installed by rbinstall on defined version
func updateGems(rubyVersion string) {
	var err error

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
		rgVersion := getAdvisableRubyGemsVersion(rubyVersion)

		spinner.Show("Updating RubyGems to %s", rgVersion)
		err = updateRubygemsTaskHandler(rubyVersion, rgVersion)
		spinner.Done(err == nil)

		if err != nil {
			terminal.PrintWarnMessage(err.Error())
		}

		installed = true
	}

	// //////////////////////////////////////////////////////////////////////////////// //

	if knf.GetS(GEMS_INSTALL) != "" {
		var gemVerInfo, installedVersion string

		for _, gem := range strings.Split(knf.GetS(GEMS_INSTALL), " ") {
			gemName, gemVersion := parseGemInfo(gem)

			if gemVersion != "" {
				gemVerInfo = fmt.Sprintf("(%s.x)", gemVersion)
			} else {
				gemVerInfo = fmt.Sprintf("(latest)")
			}

			if isGemInstalled(rubyVersion, gemName) {
				spinner.Show("Updating %s %s", gemName, gemVerInfo)
				installedVersion, err = updateGemTaskHandler(rubyVersion, gemName, gemVersion)
			} else {
				spinner.Show("Installing %s %s", gemName, gemVerInfo)
				installedVersion, err = installGemTaskHandler(rubyVersion, gemName, gemVersion)
			}

			spinner.Done(err == nil)

			if err == nil {
				if installedVersion != "" {
					log.Info(
						"[%s]Gem %s updated to version %s for %s",
						currentUser.RealName, gem, installedVersion, rubyVersion,
					)
				}
			} else {
				terminal.PrintWarnMessage(err.Error())
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
func runGemCmd(rubyVersion, cmd, gem, gemVersion string) (string, error) {
	start := time.Now()
	rubyPath := getVersionPath(rubyVersion)
	gemCmd := exec.Command(rubyPath+"/bin/ruby", rubyPath+"/bin/gem", cmd, "--force", gem)

	if gemVersion != "" {
		gemCmd.Args = append(gemCmd.Args, "--version", fmt.Sprintf("~>%s", gemVersion))
	}

	if knf.GetB(GEMS_NO_DOCUMENT) {
		gemCmd.Args = append(gemCmd.Args, "--no-document")
	}

	if knf.GetS(GEMS_SOURCE) != "" {
		gemCmd.Args = append(gemCmd.Args, "--clear-sources", "--source", getGemSourceURL(rubyVersion))
	}

	output, err := gemCmd.CombinedOutput()

	if err == nil {
		version := getInstalledGemVersion(rubyVersion, gem, start)

		if version == "" {
			return "", nil
		}

		return version, nil
	}

	if gemVersion == "" {
		gemVersion = "latest"
	} else {
		gemVersion += ".x"
	}

	actionLog, err := logFailedAction(strings.TrimRight(string(output), "\r\n"))

	if err == nil {
		switch cmd {
		case "update":
			return "", fmtc.Errorf("Can't update gem %s (%s). Gem command output saved as %s.", gem, gemVersion, actionLog)
		default:
			return "", fmtc.Errorf("Can't install gem %s (%s). Gem command output saved as %s.", gem, gemVersion, actionLog)
		}
	}

	switch cmd {
	case "update":
		return "", fmtc.Errorf("Can't update gem %s (%s)", gem, gemVersion)
	default:
		return "", fmtc.Errorf("Can't install gem %s (%s)", gem, gemVersion)
	}
}

// updateRubygems update rubygems to defined version
func updateRubygems(rubyVersion, gemVersion string) error {
	var gemCmd *exec.Cmd

	rubyPath := getVersionPath(rubyVersion)

	if gemVersion == "latest" {
		gemCmd = exec.Command(rubyPath+"/bin/ruby", rubyPath+"/bin/gem", "update", "--system")
	} else {
		gemCmd = exec.Command(rubyPath+"/bin/ruby", rubyPath+"/bin/gem", "update", "--system", gemVersion)
	}

	if knf.GetS(GEMS_SOURCE) != "" {
		gemCmd.Args = append(gemCmd.Args, "--clear-sources", "--source", getGemSourceURL(rubyVersion))
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
func downloadFile(info *index.VersionInfo) (string, error) {
	url := knf.GetS(STORAGE_URL) + "/" + info.Path + "/" + info.File
	tmpDir, err := temp.MkDir()

	if err != nil {
		return "", err
	}

	fd, err := os.OpenFile(tmpDir+"/"+info.File, os.O_CREATE|os.O_WRONLY, 0644)

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
		pb := progress.New(resp.ContentLength, "")
		pb.Start()
		_, err = io.Copy(fd, pb.Reader(resp.Body))
		pb.Finish()
	}

	return tmpDir + "/" + info.File, err
}

// printCurrentVersionName print version from given slice for
// versions listing
func printCurrentVersionName(category string, versions index.CategoryData, installed map[string]bool, counter int) bool {
	if len(versions) == 0 && counter == 0 {
		printSized(" {s-}%%-%ds{!} ", categorySize[category], NONE_VERSION)
		return true
	}

	if len(versions) <= counter {
		printSized(" %%-%ds ", categorySize[category], "")
		return false
	}

	info := versions[counter]
	prettyName := info.Name

	if strings.HasPrefix(prettyName, "2.") && strutil.Substr(prettyName, 2, 1) != "0" {
		prettyName = strutil.Exclude(prettyName, "-p0")
	}

	if strings.HasPrefix(prettyName, "3.") {
		prettyName = strutil.Exclude(prettyName, "-p0")
	}

	if len(info.Variations) > 0 {
		if isAnyVariationInstalled(info, installed) {
			prettyName += generateInstallBullets(info, installed, categoryColor[category])
		} else {
			prettyName += fmt.Sprintf(" {s-}+%d{!}", len(info.Variations))
		}
	} else {
		if installed[info.Name] {
			prettyName += " " + getInstallBullet(installed[info.Name], categoryColor[category])
		}
	}

	printRubyVersion(category, prettyName)

	return true
}

// isAnyVariationInstalled returns true if any variation of given version is installed
func isAnyVariationInstalled(info *index.VersionInfo, installed map[string]bool) bool {
	if installed[info.Name] {
		return true
	}

	for _, variation := range info.Variations {
		if installed[variation.Name] {
			return true
		}
	}

	return false
}

// generateInstallBullets generates bullets for installed versions
func generateInstallBullets(info *index.VersionInfo, installed map[string]bool, color string) string {
	result := " "

	result += getInstallBullet(installed[info.Name], color)

	for _, variation := range info.Variations {
		result += getInstallBullet(installed[variation.Name], color)
	}

	return result
}

// getInstallBullet returns install bullet with style for given version
func getInstallBullet(installed bool, color string) string {
	if installed {
		if fmtc.DisableColors {
			return "•"
		} else {
			return fmt.Sprintf("{%s}•{!}", color)
		}
	} else {
		if fmtc.DisableColors {
			return "-"
		} else {
			return "{s-}•{!}"
		}
	}
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
		categorySize[index.CATEGORY_RUBY] = DEFAULT_CATEGORY_SIZE
		categorySize[index.CATEGORY_JRUBY] = DEFAULT_CATEGORY_SIZE
		categorySize[index.CATEGORY_TRUFFLE] = DEFAULT_CATEGORY_SIZE
		categorySize[index.CATEGORY_OTHER] = DEFAULT_CATEGORY_SIZE

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

// getAdvisableRubyGemsVersion returns recommended RubyGems version for
// given version of Ruby
func getAdvisableRubyGemsVersion(rubyVersion string) string {
	if strings.HasPrefix(rubyVersion, "jruby-") {
		rubyVersion = strutil.Exclude(rubyVersion, "jruby-")

		if !strings.HasPrefix(rubyVersion, "9.2") {
			return MIN_RUBYGEMS_VERSION_JRUBY
		}

		return "latest"
	}

	v, err := version.Parse(strutil.ReadField(rubyVersion, 0, false, "-"))
	minVer, _ := version.Parse("2.3.0")

	if err != nil || v.Less(minVer) {
		return MIN_RUBYGEMS_VERSION_BASE
	}

	return "latest"
}

// getVersionInfo finds info about given version in index
func getVersionInfo(rubyVersion string) (*index.VersionInfo, string, error) {
	osName, archName, err := getSystemInfo()

	if err != nil {
		return nil, "", err
	}

	info, category := repoIndex.Find(osName, archName, rubyVersion)

	if info == nil {
		return nil, "", fmt.Errorf("Can't find info about version %s for your OS", rubyVersion)
	}

	return info, category, nil
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
func getGemSourceURL(rubyVersion string) string {
	if strings.HasPrefix(rubyVersion, "1.8") {
		return "http://" + knf.GetS(GEMS_SOURCE)
	}

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
func checkDependencies(info *index.VersionInfo, category string) {
	if category == index.CATEGORY_JRUBY && env.Which("java") == "" {
		printErrorAndExit("Java is required for this variation of Ruby")
	}

	if strings.HasSuffix(info.Name, "jemalloc") {
		if !isLibLoaded("libjemalloc.so.2") {
			printErrorAndExit("Jemalloc 5+ is required for this version of Ruby")
		}
	}
}

// getSystemInfo return info about system
func getSystemInfo() (string, string, error) {
	var os, arch string

	systemInfo, err := system.GetSystemInfo()

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

	osInfo, err := system.GetOSInfo()

	if err != nil {
		return "", "", fmt.Errorf("Can't get information about OS")
	}

	osVersion, err := version.Parse(osInfo.VersionID)

	if err != nil {
		return "", "", fmt.Errorf("Can't parse OS version")
	}

	os = fmt.Sprintf("%s-%d", osInfo.ID, osVersion.Major())

	return os, arch, nil
}

// isLibLoaded return true if given library is loaded
func isLibLoaded(glob string) bool {
	cmd := exec.Command("ldconfig", "-p")
	output, err := cmd.Output()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	for _, line := range strings.Split(string(output), "\n") {
		if !strings.Contains(line, "=>") {
			continue
		}

		line = strings.TrimSpace(line)
		line = strutil.ReadField(line, 0, false, " ")

		match, _ := filepath.Match(glob, line)

		if match {
			return true
		}
	}

	return false
}

// isVersionSupportedByBundler returns true if given version is supported by the
// latest version of bundler
func isVersionSupportedByBundler(rubyVersion string) bool {
	major := strutil.Head(rubyVersion, 1)

	if !strings.ContainsAny(major, "12") {
		return true
	}

	if major == "1" {
		return false
	}

	minor := strutil.ReadField(rubyVersion, 1, false, ".")

	if strings.ContainsAny(minor, "012") {
		return false
	}

	return true
}

// getNameWithoutPatchLevel return name without -p0
func getNameWithoutPatchLevel(name string) string {
	return strings.Replace(name, "-p0", "", -1)
}

// parseGemInfo extract name and version of gem
func parseGemInfo(data string) (string, string) {
	if !strings.Contains(data, "=") {
		return data, ""
	}

	return strutil.ReadField(data, 0, false, "="),
		strutil.ReadField(data, 1, false, "=")
}

// logFailedAction save data to temporary log file and return path
// to this log file
func logFailedAction(message string) (string, error) {
	if len(message) == 0 {
		return "", errors.New("Output data is empty")
	}

	logSuffix := passwd.GenPassword(8, passwd.STRENGTH_WEAK)
	tmpName := fmt.Sprintf("%s/rbinstall-fail-%s.log", knf.GetS(MAIN_TMP_DIR), logSuffix)

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
	spinner.Done(false)
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

// showUsage prints usage info
func showUsage() {
	genUsage().Render()
}

// showAbout prints info about version
func showAbout(gitRev string) {
	genAbout(gitRev).Render()
}

// showVerboseAbout prints verbose info about app
func showVerboseAbout(gitRev string, gomod []byte) {
	support.ShowSupportInfo(APP, VER, gitRev, gomod)
}

// genCompletion generates completion for different shells
func genCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Printf(bash.Generate(info, "rbinstall"))
	case "fish":
		fmt.Printf(fish.Generate(info, "rbinstall"))
	case "zsh":
		fmt.Printf(zsh.Generate(info, optMap, "rbinstall"))
	default:
		return 1
	}

	return 0
}

// genMan generates man page
func genMan() int {
	fmt.Println(
		man.Generate(
			genUsage(),
			genAbout(""),
		),
	)

	return 0
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo("", "version")

	info.AppNameColorTag = "{*}" + colorTagApp

	info.AddOption(OPT_REINSTALL, "Reinstall already installed version {s-}(if allowed in config){!}")
	info.AddOption(OPT_UNINSTALL, "Uninstall already installed version {s-}(if allowed in config){!}")
	info.AddOption(OPT_GEMS_UPDATE, "Update gems for some version {s-}(if allowed in config){!}")
	info.AddOption(OPT_REHASH, "Rehash rbenv shims")
	info.AddOption(OPT_GEMS_INSECURE, "Use HTTP instead of HTTPS for installing gems")
	info.AddOption(OPT_RUBY_VERSION, "Install version defined in version file")
	info.AddOption(OPT_INFO, "Print detailed info about version")
	info.AddOption(OPT_ALL, "Print all available versions")
	info.AddOption(OPT_NO_PROGRESS, "Disable progress bar and spinner")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample("2.0.0-p598", "Install 2.0.0-p598")
	info.AddExample("2.0.0", "Install latest available release in 2.0.0")
	info.AddExample("2.0.0 -i", "Show details and available variations for 2.0.0")
	info.AddExample("2.0.0-p598-railsexpress", "Install 2.0.0-p598 with railsexpress patches")
	info.AddExample("2.0.0-p598 -G", "Update gems installed for 2.0.0-p598")
	info.AddExample("2.0.0-p598 --reinstall", "Reinstall 2.0.0-p598")
	info.AddExample("-r", "Install version defined in .ruby-version file")

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	return &usage.About{
		App:           APP,
		Version:       VER,
		Desc:          DESC,
		Year:          2006,
		Owner:         "ESSENTIAL KAOS",
		License:       "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
		UpdateChecker: usage.UpdateChecker{"essentialkaos/rbinstall", update.GitHubChecker},

		AppNameColorTag: "{*}" + colorTagApp,
		VersionColorTag: colorTagVer,
	}
}

<p align="center"><a href="#readme"><img src="https://gh.kaos.st/rbinstall.svg"/></a></p>

<p align="center">
  <a href="https://github.com/essentialkaos/rbinstall/actions"><img src="https://github.com/essentialkaos/rbinstall/workflows/CI/badge.svg" alt="GitHub Actions Status" /></a>
  <a href="https://github.com/essentialkaos/rbinstall/actions?query=workflow%3ACodeQL"><img src="https://github.com/essentialkaos/rbinstall/workflows/CodeQL/badge.svg" /></a>
  <a href="https://goreportcard.com/report/github.com/essentialkaos/rbinstall"><img src="https://goreportcard.com/badge/github.com/essentialkaos/rbinstall" /></a>
  <a href="https://codebeat.co/projects/github-com-essentialkaos-rbinstall-master"><img alt="codebeat badge" src="https://codebeat.co/badges/b78de32a-6867-4bd3-9135-8244d4813531" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<p align="center">
  <a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#build-status">Build Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a>
</p>

`rbinstall` is a utility for installing prebuilt Ruby to [rbenv](https://github.com/rbenv/rbenv).

### Usage demo

[![demo](https://gh.kaos.st/rbinstall-200.gif)](#usage-demo)

### Installation

#### From [ESSENTIAL KAOS Public Repository](https://yum.kaos.st)

```bash
sudo yum install -y https://yum.kaos.st/get/$(uname -r).rpm
sudo yum install rbinstall
```

#### Using `install.sh`
We provide simple bash script `install.sh` for installing the application from the sources.

```bash
# install rbenv, golang and latest 7zip
# set GOPATH

git clone https://kaos.sh/rbinstall.git
cd rbinstall

sudo ./install.sh
```

If you have some issues with installing, try to use script in debug mode:

```bash
sudo ./install.sh --debug
```

### Usage
```
Usage: rbinstall {options} version

Options

  --reinstall, -R        Reinstall already installed version (if allowed in config)
  --uninstall, -U        Uninstall already installed version (if allowed in config)
  --gems-update, -G      Update gems for some version (if allowed in config)
  --rehash, -H           Rehash rbenv shims
  --gems-insecure, -s    Use HTTP instead of HTTPS for installing gems
  --ruby-version, -r     Install version defined in version file
  --info, -i             Print detailed info about version
  --all, -a              Print all available versions
  --no-progress, -np     Disable progress bar and spinner
  --no-color, -nc        Disable colors in output
  --help, -h             Show this help message
  --version, -v          Show version

Examples

  rbinstall 2.0.0-p598
  Install 2.0.0-p598

  rbinstall 2.0.0
  Install latest available release in 2.0.0

  rbinstall 2.0.0 -i
  Show details and available variations for 2.0.0

  rbinstall 2.0.0-p598-railsexpress
  Install 2.0.0-p598 with railsexpress patches

  rbinstall 2.0.0-p598 -G
  Update gems installed for 2.0.0-p598

  rbinstall 2.0.0-p598 --reinstall
  Reinstall 2.0.0-p598

  rbinstall -r
  Install version defined in .ruby-version file

```

### Build Status

| Branch | Status |
|--------|--------|
| `master` | [![CI](https://github.com/essentialkaos/rbinstall/workflows/CI/badge.svg?branch=master)](https://github.com/essentialkaos/rbinstall/actions) |
| `develop` | [![CI](https://github.com/essentialkaos/rbinstall/workflows/CI/badge.svg?branch=develop)](https://github.com/essentialkaos/rbinstall/actions) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>

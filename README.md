<p align="center"><a href="#readme"><img src="https://gh.kaos.st/rbinstall.svg"/></a></p>

<p align="center">
  <a href="https://travis-ci.com/essentialkaos/rbinstall"><img src="https://travis-ci.com/essentialkaos/rbinstall.svg?branch=master" /></a>
  <a href="https://goreportcard.com/report/github.com/essentialkaos/rbinstall"><img src="https://goreportcard.com/badge/github.com/essentialkaos/rbinstall" /></a>
  <a href="https://codebeat.co/projects/github-com-essentialkaos-rbinstall-master"><img alt="codebeat badge" src="https://codebeat.co/badges/b78de32a-6867-4bd3-9135-8244d4813531" /></a>
  <a href="https://essentialkaos.com/ekol"><img src="https://gh.kaos.st/ekol.svg" /></a>
</p>

<p align="center">
  <a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#build-status">Build Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a>
</p>

`rbinstall` is a utility for installing prebuilt ruby to [rbenv](https://github.com/rbenv/rbenv).

### Usage demo

[![demo](https://gh.kaos.st/rbinstall-0121.gif)](#usage-demo)

### Installation

#### From ESSENTIAL KAOS Public repo for RHEL6/CentOS6

```bash
[sudo] yum install -y https://yum.kaos.st/6/release/x86_64/kaos-repo-9.2-0.el6.noarch.rpm
[sudo] yum install rbinstall
```

#### From ESSENTIAL KAOS Public repo for RHEL7/CentOS7

```bash
[sudo] yum install -y https://yum.kaos.st/7/release/x86_64/kaos-repo-9.2-0.el7.noarch.rpm
[sudo] yum install rbinstall
```

#### Using `install.sh`
We provide simple bash script `install.sh` for installing the application from the sources.

```bash
... install rbenv, golang and latest 7zip
... set GOPATH

git clone https://github.com/essentialkaos/rbinstall.git
cd rbinstall

[sudo] ./install.sh
```

If you have some issues with installing, try to use script in debug mode:

```
[sudo] ./install.sh --debug
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
| `master` | [![Build Status](https://travis-ci.com/essentialkaos/rbinstall.svg?branch=master)](https://travis-ci.com/essentialkaos/rbinstall) |
| `develop` | [![Build Status](https://travis-ci.com/essentialkaos/rbinstall.svg?branch=develop)](https://travis-ci.com/essentialkaos/rbinstall) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[EKOL](https://essentialkaos.com/ekol)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>

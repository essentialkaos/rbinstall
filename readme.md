![RBInstall Logo](https://essentialkaos.com/github/rbinstall-v1.png)

`rbinstall` is utility for installing prebuilt ruby to RBEnv.

[Usage demo](#usage-demo) • [Installation](#installation) • [Usage](#usage) • [Build Status](#build-status) • [Contributing](#contributing) • [License](#license)

## Usage demo

[![asciicast](https://essentialkaos.com/github/rbinstall-073.gif)](https://asciinema.org/a/47983)

## Installation

#### From ESSENTIAL KAOS Public repo for RHEL6/CentOS6

```
yum install -y http://release.yum.kaos.io/i386/kaos-repo-6.8-0.el6.noarch.rpm
yum install rbinstall
```

#### Using install.sh

We provide simple bash script `script.sh` for installing app from the sources.

```
... install rbenv, golang and latest 7zip
... set GOPATH

git clone https://github.com/essentialkaos/rbinstall.git
cd rbinstall

sudo ./install.sh
```

If you have some issues with installing, try to use script in debug mode:

```
sudo ./install.sh --debug
```

## Usage
```
Usage: rbinstall <options> version

Options:

  --gems-update, -g       Update gems for some version
  --gems-insecure, -S     Use http instead https for installing gems
  --ruby-version, -r      Install version defined in version file
  --no-color, -nc         Disable colors in output
  --no-progress, -np      Disable progress bar and spinner
  --help, -h              Show this help message
  --version, -v           Show version

Examples:

  rbinstall 2.0.0-p598
  Install 2.0.0-p598

  rbinstall 2.0.0-p598-railsexpress
  Install 2.0.0-p598 with railsexpress patches

  rbinstall 2.0.0-p598 -g
  Update gems installed on 2.0.0-p598

  rbinstall -r
  Install version defined in .ruby-version file

```

## Build Status

| Branch | Status |
|------------|--------|
| `master` | [![Build Status](https://travis-ci.org/essentialkaos/rbinstall.svg?branch=master)](https://travis-ci.org/essentialkaos/rbinstall) |
| `develop` | [![Build Status](https://travis-ci.org/essentialkaos/rbinstall.svg?branch=develop)](https://travis-ci.org/essentialkaos/rbinstall) |

## Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

## License

[EKOL](https://essentialkaos.com/ekol)

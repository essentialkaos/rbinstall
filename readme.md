### RBInstall

`rbinstall` is utility for installing prebuilt ruby to RBEnv.

#### Installation

###### From ESSENTIAL KAOS Public repo for RHEL6/CentOS6

```
yum install -y http://release.yum.kaos.io/i386/kaos-repo-6.8-0.el6.noarch.rpm
yum install rbinstall
```

###### Using install.sh

We provide simple bash script `script.sh` for installing app from the sources.

```
... install rbenv, golang and 7zip
... set GOPATH

git clone https://github.com/essentialkaos/rbinstall.git
cd rbinstall
git checkout v0.5.0

sudo ./install.sh
```

If you have some issues with installing, try to use script in debug mode:

```
sudo ./install.sh --debug
```

#### Usage
```
Usage: rbinstall <options> version

Options:

  --gems-update, -g       Update gems for some version
  --gems-insecure, -S     Use http instead https for installing gems
  --no-color, -nc         Disable colors in output
  --help, -h              Show this help message
  --version, -v           Show version

Examples:

  rbinstall 2.0.0-p598
  Install 2.0.0-p598

  rbinstall 2.0.0-p598-railsexpress
  Install 2.0.0-p598 with railsexpress patches

  rbinstall 2.0.0-p598 -g
  Update gems installed on 2.0.0-p598

```

#### Build Status

| Repository | Status |
|------------|--------|
| Stable | [![Build Status](https://travis-ci.org/essentialkaos/rbinstall.svg?branch=master)](https://travis-ci.org/essentialkaos/rbinstall) |
| Unstable | [![Build Status](https://travis-ci.org/essentialkaos/rbinstall.svg?branch=develop)](https://travis-ci.org/essentialkaos/rbinstall) |

#### License

[EKOL](https://essentialkaos.com/ekol)

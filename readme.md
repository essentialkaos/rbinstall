<p align="center">
<a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#build-status">Build Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a>
</p>

<p align="center">
<img width="300" height="150" src="https://gh.kaos.io/rbinstall.png"/>
</p>

`rbinstall` is utility for installing prebuilt ruby to [rbenv](https://github.com/rbenv/rbenv).

## Usage demo

[![demo](https://gh.kaos.io/rbinstall-0100.gif)](#usage-demo)

## Installation


<details>
<summary><strong>From ESSENTIAL KAOS Public repo for RHEL6/CentOS6</strong></summary>
```
[sudo] yum install -y https://yum.kaos.io/6/release/i386/kaos-repo-7.2-0.el6.noarch.rpm
[sudo] yum install rbinstall
```
</details>

<details>
<summary><strong>From ESSENTIAL KAOS Public repo for RHEL7/CentOS7</strong></summary>
```
[sudo] yum install -y https://yum.kaos.io/7/release/x86_64/kaos-repo-7.2-0.el7.noarch.rpm
[sudo] yum install rbinstall
```
</details>

<details>
<summary><strong>Using install.sh</strong></summary>
We provide simple bash script `install.sh` for installing app from the sources.

```
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
</details>

## Usage
```
Usage: rbinstall {options} version

Options

  --gems-update, -g      Update gems for some version (if allowed in config)
  --gems-insecure, -S    Use http instead https for installing gems
  --ruby-version, -r     Install version defined in version file
  --reinstall, -R        Reinstall already installed version (if allowed in config)
  --rehash, -H           Rehash rbenv shims
  --no-color, -nc        Disable colors in output
  --no-progress, -np     Disable progress bar and spinner
  --help, -h             Show this help message
  --version, -v          Show version

Examples

  rbinstall 2.0.0-p598
  Install 2.0.0-p598

  rbinstall 2.0.0-p598-railsexpress
  Install 2.0.0-p598 with railsexpress patches

  rbinstall 2.0.0-p598 -g
  Update gems installed for 2.0.0-p598

  rbinstall 2.0.0-p598 --reinstall
  Reinstall 2.0.0-p598

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

<p align="center"><a href="#readme"><img src="https://gh.kaos.st/rbinstall.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/w/rbinstall/ci"><img src="https://kaos.sh/w/rbinstall/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/r/rbinstall"><img src="https://kaos.sh/r/rbinstall.svg" alt="GoReportCard" /></a>
  <a href="https://kaos.sh/b/rbinstall"><img src="https://kaos.sh/b/b78de32a-6867-4bd3-9135-8244d4813531.svg" alt="codebeat badge" /></a>
  <a href="https://kaos.sh/w/rbinstall/codeql"><img src="https://kaos.sh/w/rbinstall/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<p align="center">
  <a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#build-status">Build Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a>
</p>

`rbinstall` is a utility for installing prebuilt Ruby to [rbenv](https://github.com/rbenv/rbenv).

### Usage demo

[![demo](https://gh.kaos.st/rbinstall-300.gif)](#usage-demo)

### Installation

#### From [ESSENTIAL KAOS Public Repository](https://yum.kaos.st)

```bash
sudo yum install -y https://yum.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo yum install rbinstall
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
| `master` | [![CI](https://kaos.sh/w/rbinstall/ci.svg?branch=master)](https://kaos.sh/w/rbinstall/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/rbinstall/ci.svg?branch=develop)](https://kaos.sh/w/rbinstall/ci?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>

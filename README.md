<p align="center"><a href="#readme"><img src=".github/images/card.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/r/rbinstall"><img src="https://kaos.sh/r/rbinstall.svg" alt="GoReportCard" /></a>
  <a href="https://kaos.sh/l/rbinstall"><img src="https://kaos.sh/l/5680ef76d53ea9526739.svg" alt="Code Climate Maintainability" /></a>
  <a href="https://kaos.sh/y/ek"><img src="https://kaos.sh/y/3a20b5e6b6364d7ba936fb42fd5729ed.svg" alt="Codacy badge" /></a>
  <a href="https://kaos.sh/w/rbinstall/ci"><img src="https://kaos.sh/w/rbinstall/ci-push.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/w/rbinstall/codeql"><img src="https://kaos.sh/w/rbinstall/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src=".github/images/license.svg"/></a>
</p>

<p align="center">
  <a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#ci-status">CI Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a>
</p>

`rbinstall` is a utility for installing prebuilt Ruby to [rbenv](https://github.com/rbenv/rbenv).

> [!NOTE]
> Take a look at our [FAQ](https://kaos.sh/rbinstall/w/FAQ) for more information.

### Usage demo

[![demo](https://gh.kaos.st/rbinstall-300.gif)](#usage-demo)

### Installation

#### From [ESSENTIAL KAOS Public Repository](https://kaos.sh/kaos-repo)

```bash
sudo dnf install -y https://pkgs.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo dnf install rbinstall
```

### Usage

#### `rbinstall`

<p align="center"><img src=".github/images/rbinstall-usage.svg"/></p>

#### `rbinstall-clone`

<p align="center"><img src=".github/images/rbinstall-clone-usage.svg"/></p>

#### `rbinstall-gen`

<p align="center"><img src=".github/images/rbinstall-gen-usage.svg"/></p>

### CI Status

| Branch | Status |
|--------|--------|
| `master` | [![CI](https://kaos.sh/w/rbinstall/ci-push.svg?branch=master)](https://kaos.sh/w/rbinstall/ci-push?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/rbinstall/ci-push.svg?branch=develop)](https://kaos.sh/w/rbinstall/ci-push?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/.github/blob/master/CONTRIBUTING.md).

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>

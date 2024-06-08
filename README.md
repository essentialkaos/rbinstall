<p align="center"><a href="#readme"><img src="https://gh.kaos.st/rbinstall.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/r/rbinstall"><img src="https://kaos.sh/r/rbinstall.svg" alt="GoReportCard" /></a>
  <a href="https://kaos.sh/l/rbinstall"><img src="https://kaos.sh/l/5680ef76d53ea9526739.svg" alt="Code Climate Maintainability" /></a>
  <a href="https://kaos.sh/b/rbinstall"><img src="https://kaos.sh/b/b78de32a-6867-4bd3-9135-8244d4813531.svg" alt="codebeat badge" /></a>
  <a href="https://kaos.sh/w/rbinstall/ci"><img src="https://kaos.sh/w/rbinstall/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/w/rbinstall/codeql"><img src="https://kaos.sh/w/rbinstall/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
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
sudo yum install -y https://pkgs.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo yum install rbinstall
```

### Usage

<img src=".github/image/usage.svg" />

### CI Status

| Branch | Status |
|--------|--------|
| `master` | [![CI](https://kaos.sh/w/rbinstall/ci.svg?branch=master)](https://kaos.sh/w/rbinstall/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/rbinstall/ci.svg?branch=develop)](https://kaos.sh/w/rbinstall/ci?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>

################################################################################

# rpmbuilder:relative-pack true

################################################################################

%define  debug_package %{nil}

################################################################################

%define _posixroot        /
%define _root             /root
%define _bin              /bin
%define _sbin             /sbin
%define _srv              /srv
%define _home             /home
%define _opt              /opt
%define _lib32            %{_posixroot}lib
%define _lib64            %{_posixroot}lib64
%define _libdir32         %{_prefix}%{_lib32}
%define _libdir64         %{_prefix}%{_lib64}
%define _logdir           %{_localstatedir}/log
%define _rundir           %{_localstatedir}/run
%define _lockdir          %{_localstatedir}/lock/subsys
%define _cachedir         %{_localstatedir}/cache
%define _spooldir         %{_localstatedir}/spool
%define _crondir          %{_sysconfdir}/cron.d
%define _loc_prefix       %{_prefix}/local
%define _loc_exec_prefix  %{_loc_prefix}
%define _loc_bindir       %{_loc_exec_prefix}/bin
%define _loc_libdir       %{_loc_exec_prefix}/%{_lib}
%define _loc_libdir32     %{_loc_exec_prefix}/%{_lib32}
%define _loc_libdir64     %{_loc_exec_prefix}/%{_lib64}
%define _loc_libexecdir   %{_loc_exec_prefix}/libexec
%define _loc_sbindir      %{_loc_exec_prefix}/sbin
%define _loc_bindir       %{_loc_exec_prefix}/bin
%define _loc_datarootdir  %{_loc_prefix}/share
%define _loc_includedir   %{_loc_prefix}/include
%define _loc_mandir       %{_loc_datarootdir}/man
%define _rpmstatedir      %{_sharedstatedir}/rpm-state
%define _pkgconfigdir     %{_libdir}/pkgconfig

################################################################################

Summary:         Utility for installing prebuilt Ruby to rbenv
Name:            rbinstall
Version:         2.3.0
Release:         0%{?dist}
Group:           Applications/System
License:         Apache License, Version 2.0
URL:             https://kaos.sh/rbinstall

Source0:         https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

BuildRoot:       %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

Requires:        rbenv ca-certificates p7zip >= 16

BuildRequires:   golang >= 1.15

Provides:        %{name} = %{version}-%{release}

################################################################################

%description
Utility for installing different prebuilt versions of Ruby to rbenv.

################################################################################

%package gen

Summary:         Utility for generating RBInstall index
Version:         2.1.0
Release:         0%{?dist}
Group:           Development/Tools

%description gen
Utility for generating RBInstall index.

################################################################################

%package clone

Summary:         Utility for cloning RBInstall repository
Version:         2.0.0
Release:         0%{?dist}
Group:           Development/Tools

%description clone
Utility for cloning RBInstall repository.

################################################################################

%prep
%setup -q

%build
export GOPATH=$(pwd)
export GO111MODULE=auto
pushd src/github.com/essentialkaos/%{name}/
%{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -dm 755 %{buildroot}%{_sysconfdir}
install -dm 755 %{buildroot}%{_logdir}
install -dm 755 %{buildroot}%{_logdir}/%{name}

install -pm 755 src/github.com/essentialkaos/%{name}/%{name} \
                %{buildroot}%{_bindir}/
install -pm 755 src/github.com/essentialkaos/%{name}/%{name}-gen \
                %{buildroot}%{_bindir}/
install -pm 755 src/github.com/essentialkaos/%{name}/%{name}-clone \
                %{buildroot}%{_bindir}/

install -pm 644 src/github.com/essentialkaos/%{name}/common/%{name}.knf \
                %{buildroot}%{_sysconfdir}/

%clean
rm -rf %{buildroot}

################################################################################

%files
%defattr(-,root,root,-)
%doc LICENSE
%dir %{_logdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}.knf
%{_bindir}/%{name}

%files gen
%defattr(-,root,root,-)
%doc LICENSE
%{_bindir}/%{name}-gen

%files clone
%defattr(-,root,root,-)
%doc LICENSE
%{_bindir}/%{name}-clone

################################################################################

%changelog
* Sat Mar 20 2021 Anton Novojilov <andy@essentialkaos.com> - 2.3.0-0
- [cli] UI improvements

* Tue Feb 09 2021 Anton Novojilov <andy@essentialkaos.com> - 2.2.0-0
- [cli|gen] Added support of Ruby 3

* Sat Jun 20 2020 Anton Novojilov <andy@essentialkaos.com> - 2.1.0-0
- [cli] Improved UI

* Wed May 20 2020 Anton Novojilov <andy@essentialkaos.com> - 2.0.0-0
- [cli] Improved UI
- [cli] Removed REE and Rubinius support
- [cli] Added TruffleRuby support

* Tue May 19 2020 Anton Novojilov <andy@essentialkaos.com> - 1.0.0-0
- [cli|gen|clone] Migrated to ek.v12
- [cli] Using zip7 package instead z7
- [cli] Improved UI

* Thu Jan 16 2020 Anton Novojilov <andy@essentialkaos.com> - 0.22.0-0
- [cli] Improved RubyGems update mechanic

* Fri Aug 16 2019 Anton Novojilov <andy@essentialkaos.com> - 0.21.5-0
- [cli] Always use insecure source for 1.8.x

* Fri Aug 16 2019 Anton Novojilov <andy@essentialkaos.com> - 0.21.4-0
- [cli] Disabled installation of the latest version of bundler gem for old
  versions of Ruby

* Thu Aug 15 2019 Anton Novojilov <andy@essentialkaos.com> - 0.21.3-0
- [cli] Improved gems update/install mechanic

* Fri Mar 22 2019 Anton Novojilov <andy@essentialkaos.com> - 0.21.2-0
- [cli] Improved jemalloc availability check

* Tue Mar 19 2019 Anton Novojilov <andy@essentialkaos.com> - 0.21.1-0
- [cli] Fixed bug with railsexpress availability info in versions listing

* Thu Mar 14 2019 Anton Novojilov <andy@essentialkaos.com> - 0.21.0-0
- [cli] New RubyGems update mechanics
- [cli] Added option -i/--info for viewing detailed information about version
- [cli|gen] Added support for jemalloc variation

* Tue Mar 05 2019 Anton Novojilov <andy@essentialkaos.com> - 0.20.2-0
- [cli] Fixed bug with tasks hanging

* Wed Feb 20 2019 Anton Novojilov <andy@essentialkaos.com> - 0.20.1-0
- [cli] Fixed bug with updating versioned gems

* Wed Feb 06 2019 Anton Novojilov <andy@essentialkaos.com> - 0.20.0-0
- [cli] Added possibility to define versions for required gems
- [cli] Gem installation error now is not critical

* Tue Jan 22 2019 Anton Novojilov <andy@essentialkaos.com> - 0.19.3-0
- [cli|gen|clone] ek package updated to v10
- [cli] z7 package updated to v8

* Fri Oct 19 2018 Anton Novojilov <andy@essentialkaos.com> - 0.19.2-0
- [cli] Minor UI improvements

* Thu May 03 2018 Anton Novojilov <andy@essentialkaos.com> - 0.19.1-0
- [cli] Possible fixed bug with spinner for fast tasks
- [cli] Minor UI improvements

* Thu Apr 26 2018 Anton Novojilov <andy@essentialkaos.com> - 0.19.0-0
- [cli|gen] Added EOL info support (end-of-life)
- [cli|gen|clone] Fixed bug with error output to stdout
- [cli|gen|clone] Code refactoring
- [cli] Minor UI improvements

* Tue Apr 24 2018 Anton Novojilov <andy@essentialkaos.com> - 0.18.1-0
- Fixed bug with using option '--no-document' for old rubygem versions
- ek package updated to latest stable release
- z7 package updated to v7

* Fri Feb 02 2018 Anton Novojilov <andy@essentialkaos.com> - 0.18.0-1
- Migrated from kaos.io to kaos.st

* Fri Jan 19 2018 Anton Novojilov <andy@essentialkaos.com> - 0.18.0-0
- [cli] Added ability to delete some ruby version
- [cli] Added error messages about used conflicts options
- [cli] Improved UI
- [cli|gen|clone] ek package updated to latest version

* Sun Dec 31 2017 Anton Novojilov <andy@essentialkaos.com> - 0.17.2-0
- [cli] Minor UI improvements

* Mon Nov 13 2017 Anton Novojilov <andy@essentialkaos.com> - 0.17.1-0
- [cli] Fixed bug with updating RubyGems gems for old Ruby
  versions (<= 1.9.3-p551)

* Tue Nov 07 2017 Anton Novojilov <andy@essentialkaos.com> - 0.17.0-0
- [cli] Fixed bug with updating gems with empty gem list
- [cli] Now required version of rubygems gem can be defined through
  configuration file
- [cli] 'gems:no-ri' and 'gems:no-rdoc' options replaced by 'gems:no-document'
- [cli] Minor UI improvements
- [cli] Code refactoring

* Wed Oct 11 2017 Anton Novojilov <andy@essentialkaos.com> - 0.16.1-0
- [cli] Fixed output for 'rbenv rehash' errors
- [cli] Improved commands errors logging

* Fri Aug 04 2017 Anton Novojilov <andy@essentialkaos.com> - 0.16.0-0
- [cli] Added rehash support for uninitialized rbenv
- [cli] Checking Ruby binary after unpacking
- [cli|gen|clone] ek package updated to latest version

* Thu May 25 2017 Anton Novojilov <andy@essentialkaos.com> - 0.15.0-0
- [cli|gen|clone] ek package updated to v9
- [cli] z7 package updated to v6

* Thu Apr 20 2017 Anton Novojilov <andy@essentialkaos.com> - 0.14.1-0
- [cli] Typo fixed
- [cli|gen|clone] Added build tags

* Sun Apr 16 2017 Anton Novojilov <andy@essentialkaos.com> - 0.14.0-0
- [cli|gen|clone] ek package updated to v8
- [cli] z7 package updated to v5

* Wed Apr 12 2017 Anton Novojilov <andy@essentialkaos.com> - 0.13.1-0
- [cli] Minor improvement in config validation

* Thu Mar 30 2017 Anton Novojilov <andy@essentialkaos.com> - 0.13.0-0
- [cli] Added support for names without patch level
- [cli] Automatic aliases creation for versions which contains -p0 in the name
- [cli|gen] Minor improvements

* Wed Mar 15 2017 Anton Novojilov <andy@essentialkaos.com> - 0.12.1-1
- [cli] Using HTTP instead of HTTPS by default

* Sat Mar 11 2017 Anton Novojilov <andy@essentialkaos.com> - 0.12.1-0
- [cli] Minor help content improvement

* Fri Mar 10 2017 Anton Novojilov <andy@essentialkaos.com> - 0.12.0-0
- [cli|gen|clone] EK package updated to v7
- [cli] z7 package updated to v4

* Mon Jan 30 2017 Anton Novojilov <andy@essentialkaos.com> - 0.11.0-0
- HTTP proxy configuration in config file

* Tue Dec 20 2016 Anton Novojilov <andy@essentialkaos.com> - 0.10.0-0
- [cli] Rubygems update feature
- [cli] Fixed colors disabling in tmux/screen

* Tue Dec 13 2016 Anton Novojilov <andy@essentialkaos.com> - 0.9.2-0
- [cli] Fixed progress bar UI
- [cli] Fixed searching OS version info

* Sat Dec 10 2016 Anton Novojilov <andy@essentialkaos.com> - 0.9.1-0
- [cli] gopkg.in replaced by pkg.re for pb package
- Fixed deps in install script

* Mon Dec 05 2016 Anton Novojilov <andy@essentialkaos.com> - 0.9.0-0
- [cli|gen|clone] Added name alias support to index
- [cli] Fixed panic in unpack task handler
- [cli] Fixed bug with listing versions
- [cli] Show listing in raw format if output is not a tty

* Sun Oct 16 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.4-0
- [cli] Fixed minor bug with rendering task symbols in some terminals

* Tue Oct 11 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.3-0
- [cli|gen|clone] EK package updated to v5

* Mon Sep 12 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.2-0
- [cli|gen|clone] Minor UI changes

* Wed Jul 27 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.1-0
- [cli] Improved installed version markers for output without colors

* Tue Jul 26 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.0-2
- [cli|gen|clone] EK package updated to latest version

* Mon Jul 25 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.0-1
- [cli|gen|clone] EK package updated to 3.1.2 with MacOS X compatibility bug

* Thu Jul 21 2016 Anton Novojilov <andy@essentialkaos.com> - 0.8.0-0
- [cli|gen|clone] Mutil OS support
- [cli] Checking java before JRuby installation
- [cli] Checking installed rbenv before install
- [cli] Improved listing (separate markers for base and railsexpress version)
- [cli|gen|clone] Migrated to ek package v3
- [gen] Fixed bug with index generation without previous version of index
- [gen] Fixed bug with processing railsexpress versions
- [gen] Fixed minor UI glitches

* Thu Jul 14 2016 Anton Novojilov <andy@essentialkaos.com> - 0.7.4-0
- [cli] Checking java before JRuby installation
- [cli] Checking installed and initialized rbenv before install
- [cli] Improved listing (separate markers for base and railsexpress
  version)
- [cli] Migrated to ek v3

* Mon Jun 06 2016 Anton Novojilov <andy@essentialkaos.com> - 0.7.3-0
- [cli] Fixed listing on small screens (< 140 symbols)

* Wed Jun 01 2016 Anton Novojilov <andy@essentialkaos.com> - 0.7.2-0
- [cli] Added argument --no-progress for disabling progress
  bar and spinner

* Tue May 24 2016 Anton Novojilov <andy@essentialkaos.com> - 0.7.1-0
- [cli] UI improvemeynts

* Fri May 13 2016 Anton Novojilov <andy@essentialkaos.com> - 0.7.0-0
- [cli] Marking installed versions in listing
- [cli] Code refactoring
- [gen] Code refactoring
- [clone] Code refactoring

* Fri May 13 2016 Anton Novojilov <andy@essentialkaos.com> - 0.6.4-0
- [cli|gen] Added index sorting
- [gen] GOMAXPROCS set to 1
- [clone] GOMAXPROCS set to 1

* Thu May 05 2016 Anton Novojilov <andy@essentialkaos.com> - 0.6.3-0
- [cli] Using real user uid/gid for fail log file

* Thu Apr 28 2016 Anton Novojilov <andy@essentialkaos.com> - 0.6.2-0
- [cli] Fixed availabile versions listing without root privileges
- [cli] z7.v2 package usage

* Mon Apr 25 2016 Anton Novojilov <andy@essentialkaos.com> - 0.6.1-0
- Added rbinstall-clone Utility for cloning RBInstall repositories
- Code refactoring

* Sat Apr 23 2016 Anton Novojilov <andy@essentialkaos.com> - 0.6.0-0
- Installing version defined in version file
- GOMAXPROCS set to 2
- Ctrl+C interception
- Code refactoring

* Fri Apr 08 2016 Anton Novojilov <andy@essentialkaos.com> - 0.5.0-0
- Improved UI
- Code refactoring

* Tue Mar 01 2016 Anton Novojilov <andy@essentialkaos.com> - 0.4.2-0
- Improved rbinstall-gen UI

* Fri Jan 22 2016 Anton Novojilov <andy@essentialkaos.com> - 0.4.1-0
- Improved gem installing/updating

* Sun Dec 27 2015 Anton Novojilov <andy@essentialkaos.com> - 0.4.0-0
- Code refactoring
- Minor improvements
- pkg.re usage for sources

* Wed Dec 02 2015 Anton Novojilov <andy@essentialkaos.com> - 0.3.5-0
- Added logging for failed actions
- Verbose error output

* Thu Nov 26 2015 Anton Novojilov <andy@essentialkaos.com> - 0.3.2-1
- Rebuilt with latest version of ek packages with some fixes

* Fri Nov 20 2015 Anton Novojilov <andy@essentialkaos.com> - 0.3.2-0
- Added support for using insecure gem sources

* Mon Nov 09 2015 Anton Novojilov <andy@essentialkaos.com> - 0.3.0-0
- Added old index reusage in rbinstall-make
- Improved gems installing and updating

* Tue Oct 13 2015 Anton Novojilov <andy@essentialkaos.com> - 0.2.5-0
- Improved error handling
- Minor improvements

* Tue Sep 22 2015 Anton Novojilov <andy@essentialkaos.com> - 0.2.4-0
- Fixed bug with checking user privileges

* Tue Sep 22 2015 Anton Novojilov <andy@essentialkaos.com> - 0.2.3-0
- Code refactoring
- Small improvements

* Fri Sep 11 2015 Anton Novojilov <andy@essentialkaos.com> - 0.2.2-0
- Small improvements
- Rebuilt with golang 1.5

* Mon Aug 31 2015 Anton Novojilov <andy@essentialkaos.com> - 0.2.1-0
- Added argument for disabling colored output

* Sat Aug 29 2015 Anton Novojilov <andy@essentialkaos.com> - 0.2-0
- Added argument for gems update
- Added actions logging
- Bugfixes and improvements

* Thu Aug 27 2015 Anton Novojilov <andy@essentialkaos.com> - 0.1.3-0
- Fixed but with checking config

* Thu Aug 27 2015 Anton Novojilov <andy@essentialkaos.com> - 0.1.2-0
- Listing now not require root privileges
- Improved help output

* Thu Aug 27 2015 Anton Novojilov <andy@essentialkaos.com> - 0.1.1-0
- Fixed rubinius group name

* Tue Aug 25 2015 Anton Novojilov <andy@essentialkaos.com> - 0.1-0
- Initial build

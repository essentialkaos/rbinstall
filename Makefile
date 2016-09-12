########################################################################################

DESTDIR?=
PREFIX?=/usr

########################################################################################

.PHONY = all clean install uninstall deps

########################################################################################

all: rbinstall rbinstall-gen rbinstall-clone

deps:
	go get -v pkg.re/essentialkaos/ek.v3
	go get -v pkg.re/essentialkaos/z7.v2
	go get -v pkg.re/essentialkaos/go-linenoise.v2
	go get -v github.com/cheggaaa/pb

rbinstall:
	go build rbinstall.go

rbinstall-gen:
	go build rbinstall-gen.go

rbinstall-clone:
	go build rbinstall-clone.go

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	mkdir -p $(DESTDIR)/etc
	cp rbinstall $(DESTDIR)$(PREFIX)/bin/
	cp rbinstall-gen $(DESTDIR)$(PREFIX)/bin/
	cp rbinstall-clone $(DESTDIR)$(PREFIX)/bin/
	cp common/rbinstall.conf $(DESTDIR)/etc/

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall-gen
	rm -f $(DESTDIR)$(PREFIX)/bin/rbinstall-clone
	rm -f $(DESTDIR)/etc/rbinstall.conf

clean:
	rm -f rbinstall
	rm -f rbinstall-gen
	rm -f rbinstall-clone

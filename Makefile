.PHONY: all clean install uninstall listfiles

BINARY := listfiles

UNAME_S := $(shell uname -s)

GOFLAGS := -mod=vendor -v -trimpath -buildmode=pie

ifeq ($(UNAME_S),Linux)
   ifneq ($(wildcard /etc/arch-release),)
       PREFIX ?= /usr
   else
       PREFIX ?= /usr/local
   endif
else ifeq ($(UNAME_S),FreeBSD)
   PREFIX ?= /usr/local
else ifeq ($(UNAME_S),Darwin)
   PREFIX ?= /usr/local
endif

BINDIR ?= $(PREFIX)/bin

all: $(BINARY)

$(BINARY):
	go build $(GOFLAGS) -o $(BINARY)

install: $(BINARY)
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)

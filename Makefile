BINARY := pal

UNAME_S := $(shell uname -s)
CGO_ENABLED ?= 1

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

.PHONY: all clean install uninstall

all: $(BINARY)

$(BINARY):
	CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -o $(BINARY)

install: $(BINARY)
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)

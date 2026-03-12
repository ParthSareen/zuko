PREFIX ?= /usr/local

.PHONY: build install uninstall

build:
	go build -o zuko .

install: build
	install -m 755 zuko $(PREFIX)/bin/zuko
	@echo ""
	@echo "Installed zuko to $(PREFIX)/bin/zuko"
	@echo "Next steps:"
	@echo "  zuko setup              # discover binaries and create shims"
	@echo "  zuko init shell         # add shims to your PATH"
	@echo "  zuko init openclaw      # or configure openclaw only"

uninstall:
	rm -f $(PREFIX)/bin/zuko
	@echo "Removed zuko from $(PREFIX)/bin"

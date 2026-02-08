.PHONY: build install clean run

BINARY_NAME=lazyrmss
INSTALL_PATH=$(HOME)/.local/bin
GO := $(shell command -v go 2>/dev/null)

build:
ifndef GO
	$(error "go is not installed. Install it from https://go.dev/dl/")
endif
	go build -o $(BINARY_NAME) .

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY_NAME) $(INSTALL_PATH)/

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

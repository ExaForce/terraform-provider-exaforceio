BINARY     = terraform-provider-exaforceio
INSTALL_DIR = ~/.terraform.d/plugins/registry.terraform.io/exaforce/exaforceio/1.0.0/$(shell go env GOOS)_$(shell go env GOARCH)

.PHONY: build install fmt lint tidy

build:
	go build -o $(BINARY) .

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/

fmt:
	gofmt -w .

lint:
	go vet ./...

tidy:
	go mod tidy

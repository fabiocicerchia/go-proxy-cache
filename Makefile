#                                                                         __
# .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
# |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
# |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
# |_____|            |__|                   |_____|
#
# Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
# Repo: https://github.com/fabiocicerchia/go-proxy-cache


.PHONY: test changelog
.SILENT: help
default: help

################################################################################
# HELP
################################################################################

help: ## prints this help
	echo "                                                                         __"
	echo " .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----."
	echo " |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|"
	echo " |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|"
	echo " |_____|            |__|                   |_____|"
	echo ""
	echo "Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License"
	echo "Repo: https://github.com/fabiocicerchia/go-proxy-cache"
	echo ""
	@gawk 'BEGIN { \
		FS = ":.*##"; \
		printf "Use: make \033[36m<target>\033[0m\n"; \
	} /^\$$?\(?[a-zA-Z_-]+\)?:.*?##/ { \
		printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 \
	} /^##@/ { \
		printf "\n\033[1m%s\033[0m\n", substr($$0, 5) \
	}' $(MAKEFILE_LIST)

################################################################################
##@ BUILD
################################################################################

build: ## build
	go build -race -o go-proxy-cache main.go

################################################################################
##@ SCA
################################################################################

sca: lint sec fmt staticcheck tlsfuzzer ## sca checks

lint: ## lint
	docker run --rm -v $$PWD:/app -w /app golangci/golangci-lint:v1.27.0 golangci-lint run -v ./...

sec: ## security scan
	which gosec || curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
	gosec ./...

fmt: ## format code
	gofmt -w -s .

staticcheck: ## staticcheck
	which staticcheck || go get honnef.co/go/tools/cmd/staticcheck
	staticcheck ./...

tlsfuzzer: ## tlsfuzzer
	pip3 install --pre tlslite-ng\n
	git clone https://github.com/tlsfuzzer/tlsfuzzer.git\n
	cd tlsfuzzer
	git clone https://github.com/warner/python-ecdsa .python-ecdsa
	ln -s .python-ecdsa/src/ecdsa/ ecdsa\n
	git clone https://github.com/tlsfuzzer/tlslite-ng .tlslite-ng\n
	ln -s .tlslite-ng/tlslite/ tlslite\n
	echo "127.0.0.1 www.w3.org" >> /etc/hosts
	PYTHONPATH=. python scripts/test-bleichenbacher-workaround.py -h www.w3.org -p 443

################################################################################
##@ TEST
################################################################################

test: test-unit test-functional test-endtoend ## test

test-unit: ## test unit
	go test -race -count=1 --tags=unit ./...

test-functional: ## test functional
	go test -race -count=1 --tags=functional ./...

test-endtoend: ## test endtoend
	go test -race -count=1 --tags=endtoend ./...

cover: ## coverage
	go test -race -count=1 --tags=all -coverprofile cover.out ./...
	go tool cover -func=cover.out
	go tool cover -html=cover.out

codecov: ## codecov
	curl -s https://codecov.io/bash | bash

################################################################################
##@ UTILITY
################################################################################

changelog: ## generate a changelog
	which gitchangelog || curl -sSL https://raw.githubusercontent.com/vaab/gitchangelog/master/src/gitchangelog/gitchangelog.py > /usr/local/bin/gitchangelog && chmod +x /usr/local/bin/gitchangelog
	gitchangelog > CHANGELOG.md

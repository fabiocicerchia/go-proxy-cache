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
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s latest
	./bin/gosec ./...

fmt: ## format code
	gofmt -w -s .

staticcheck: ## staticcheck
	wget https://github.com/dominikh/go-tools/releases/download/2020.1.6/staticcheck_linux_amd64.tar.gz
	tar xvzf staticcheck_linux_amd64.tar.gz
	./staticcheck/staticcheck ./...

tlsfuzzer: ## tlsfuzzer
	go run main.go &
	echo "127.0.0.1 www.w3.org" | sudo tee -a /etc/hosts
	pip3 install --pre tlslite-ng
	git clone https://github.com/tlsfuzzer/tlsfuzzer
	cd tlsfuzzer; \
	git clone https://github.com/warner/python-ecdsa .python-ecdsa; \
	ln -s .python-ecdsa/src/ecdsa/ ecdsa; \
	git clone https://github.com/tlsfuzzer/tlslite-ng .tlslite-ng; \
	ln -s .tlslite-ng/tlslite/ tlslite; \
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

cover:  ## coverage
	go test -race -count=1 --tags=all -coverprofile c.out ./...
	go tool cover -func=c.out
	go tool cover -html=c.out

codeclimate:  ## codeclimate
	wget -O test-reporter https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64 && chmod +x test-reporter
	./test-reporter before-build
	make cover
	./test-reporter --debug after-build

codecov: ## codecov
	curl -s https://codecov.io/bash | bash

################################################################################
##@ UTILITY
################################################################################

changelog: ## generate a changelog
	which gitchangelog || curl -sSL https://raw.githubusercontent.com/vaab/gitchangelog/master/src/gitchangelog/gitchangelog.py > /usr/local/bin/gitchangelog && chmod +x /usr/local/bin/gitchangelog
	gitchangelog > CHANGELOG.md

#                                                                         __
# .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
# |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
# |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
# |_____|            |__|                   |_____|
#
# Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
# Repo: https://github.com/fabiocicerchia/go-proxy-cache

IS_LINUX=$(shell (ls -1 /etc/issue || true) | wc -l | awk '{$$1=$$1;print}')

.PHONY: test changelog staticcheck tlsfuzzer
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
	echo "Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License"
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
	go build -o go-proxy-cache main.go

build-race: ## build-race
	go build -race -o go-proxy-cache main.go

build-multiarch: ## build-multiarch
	./bin/build-multiarch.sh

################################################################################
##@ SCA
################################################################################

sca: lint sec fmt staticcheck tlsfuzzer ## sca checks

lint: ## lint
	docker run --rm -v $$PWD:/app -w /app golangci/golangci-lint:v1.42.0 golangci-lint run -v ./...

sec: ## security scan
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s latest
	./bin/gosec ./...

fmt: ## format code
	gofmt -w -s .

staticcheck: ## staticcheck
ifeq ($(IS_LINUX),1)
	wget -O staticcheck_amd64.tar.gz https://github.com/dominikh/go-tools/releases/download/2021.1.1/staticcheck_linux_amd64.tar.gz
else
	wget -O staticcheck_amd64.tar.gz https://github.com/dominikh/go-tools/releases/download/2021.1.1/staticcheck_darwin_amd64.tar.gz
endif
	tar xvzf staticcheck_amd64.tar.gz
	chmod +x ./staticcheck/staticcheck
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

test: test-unit test-functional test-endtoend test-ws test-http2 ## test

test-unit: ## test unit
	TESTING=1 go test -v -race -count=1 --tags=unit ./...

test-functional: ## test functional
	python3 -m http.server &> /dev/null &
	TESTING=1 go test -v -race -count=1 --tags=functional ./...
	pgrep python3 | xargs kill || true

test-endtoend: ## test endtoend
	go test -v -race -count=1 --tags=endtoend ./...

test-ws: ## test websocket
	cd test/full-setup/ws && npm install
	node test/full-setup/ws/ws_client.js

# @DEPRECATED
test-http2: ## test HTTP2
	nghttp -ans https://testing.local:50443/push

cover:  ## coverage
	python3 -m http.server &> /dev/null &
	TESTING=1 go test -race -count=1 --tags=unit,functional -coverprofile c.out ./...
	go tool cover -func=c.out
	go tool cover -html=c.out
	pgrep python3 | xargs kill || true

codeclimate:  ## codeclimate
ifeq ($(IS_LINUX),1)
	wget -O staticcheck_amd64.tar.gz https://github.com/dominikh/go-tools/releases/download/2021.1.1/staticcheck_linux_amd64.tar.gz
else
	wget -O test-reporter https://codeclimate.com/downloads/test-reporter/test-reporter-latest-darwin-amd64 && chmod +x test-reporter
endif

# CODECLIMATE WORKAROUND
	mkdir -p ./github.com/fabiocicerchia
	ln -s $$PWD ./github.com/fabiocicerchia/go-proxy-cache
# / CODECLIMATE WORKAROUND

	./test-reporter before-build
	make cover
	./test-reporter --debug after-build

codecov: ## codecov
	curl -s https://codecov.io/bash | bash

################################################################################
##@ UTILITY
################################################################################

.install-changelog:
	pip3 install gitchangelog pystache

changelog: .install-changelog ## generate a changelog
	which gitchangelog || curl -sSL https://raw.githubusercontent.com/vaab/gitchangelog/master/src/gitchangelog/gitchangelog.py > /usr/local/bin/gitchangelog && chmod +x /usr/local/bin/gitchangelog
	gitchangelog > CHANGELOG.md
	cat CHANGELOG.md | awk 'BEGIN {RS=""}{gsub(/^\*/,"-")}1' | tee CHANGELOG.md
	markdownlint --fix CHANGELOG.md || true

release: ## release
	cat main.go | sed "s/const AppVersion = .*/const AppVersion = \"$$VER\"/" | tee main.go
	make changelog
	git add CHANGELOG.md
	git commit -m "updated changelog for v$$VER"
	git tag -af v$$VER -m "Release v$$VER"

docker-push: ## build and push a docker image
	docker build -t fabiocicerchia/go-proxy-cache:latest -t fabiocicerchia/go-proxy-cache:$$VER .
	docker push fabiocicerchia/go-proxy-cache:latest
	docker push fabiocicerchia/go-proxy-cache:$$VER

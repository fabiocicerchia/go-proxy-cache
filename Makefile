#                                                                         __
# .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
# |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
# |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
# |_____|            |__|                   |_____|
#
# Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
# Repo: https://github.com/fabiocicerchia/go-proxy-cache

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
		printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 \
	} /^##@/ { \
		printf "\n\033[1m%s\033[0m\n", substr($$0, 5) \
	}' $(MAKEFILE_LIST)

################################################################################
##@ BUILD
################################################################################

build: ## build
	go build -o go-proxy-cache main.go

################################################################################
##@ TEST
################################################################################

test: ## test
	go test ./...

cover: ## coverage
	go test -coverprofile cover.out ./...
	go tool cover -html=cover.out

codecov: ## codecov
	curl -s https://codecov.io/bash | bash

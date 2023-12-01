SHELL := bash
VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)

REPOS ?= tyk tyk-analytics tyk-pump tyk-identity-broker tyk-sink portal
GITHUB_TOKEN ?= $(shell pass me/github)

gromit: clean */*.go confgen/templates/* policy/templates/* policy/prs/*
	! ls **/#*#
	diff policy/templates/releng/.github/workflows/release.yml.d/api-tests.gotmpl policy/templates/api-tests/.github/workflows/api-tests.yml.d/api-tests.gotmpl || cp -v policy/templates/releng/.github/workflows/release.yml.d/api-tests.gotmpl policy/templates/api-tests/.github/workflows/api-tests.yml.d/api-tests.gotmpl
	go build -v -trimpath -ldflags "-X github.com/TykTechnologies/gromit/util.version=$(VERSION) -X github.com/TykTechnologies/gromit/util.commit=$(COMMIT) -X github.com/TykTechnologies/gromit/util.buildDate=$(BUILD_DATE)"
	go mod tidy

test: 
	echo Use a config file locally and env variables in CI
	go test -coverprofile cp.out ./... # dlv test ./cmd #

update-test-cases:
	echo Updating test cases for cmd test
	go test ./cmd/ -update

clean:
	find . -name rice-box.go | xargs rm -fv
	rm -fv gromit

sync: gromit
	@$(foreach r,$(REPOS), GITHUB_TOKEN=$(GITHUB_TOKEN) ./gromit policy sync $(r);)

.PHONY: clean update-test-cases test

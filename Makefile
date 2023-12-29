SHELL := bash
VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)
SRC := $(shell find cmd confgen config orgs policy -regextype egrep -regex '.*(go|(go)?tmpl|ya?ml)')

REPOS ?= tyk tyk-analytics tyk-pump tyk-identity-broker tyk-sink portal
GITHUB_TOKEN ?= $(shell pass me/github)


gromit: *.go $(SRC)
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
	rm -rf $(REPOS)
	rm -fv gromit

sync: clean gromit
	@$(foreach r,$(REPOS), GITHUB_TOKEN=$(GITHUB_TOKEN) ./gromit policy sync $(r);)

loc: clean
	gocloc --skip-duplicated --not-match-d=\.terraform --output-type=json ~gromit ~ci | jq -r '.languages | map([.name, .code]) | transpose[] | @csv'

.PHONY: clean update-test-cases test loc

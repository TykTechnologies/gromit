SHELL 	:= bash
VERSION := $(shell git describe --tags)
COMMIT 	:= $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)

SRC 	     := $(shell find cmd config confgen env orgs policy util -name '*.go')
TEMPLATES    := $(shell find policy/{templates,app} -type f)

REPOS        ?= tyk tyk-analytics tyk-pump tyk-identity-broker tyk-sink portal tyk-pro
GITHUB_TOKEN ?= $(shell pass me/github)
JIRA_USER    ?= alok@tyk.io
JIRA_TOKEN   ?= $(shell pass Tyk/atlassian)
VARIATION    ?= prod-variation
PC_TOKEN     ?= $(shell pass Tyk/packagecloud)

UNSTABLE_REPOS := tyk-gateway-unstable tyk-dashboard-unstable tyk-pump-unstable tyk-mdcb-unstable portal-unstable tyk-identity-broker-unstable tyk-sync-unstable
STABLE_REPOS := tyk-gateway tyk-dashboard tyk-pump tyk-mdcb portal tyk-identity-broker

gromit: go.mod go.sum *.go $(SRC) $(TEMPLATES) update-variation
	go build -v -trimpath -ldflags "-X github.com/TykTechnologies/gromit/util.version=$(VERSION) -X github.com/TykTechnologies/gromit/util.commit=$(COMMIT) -X github.com/TykTechnologies/gromit/util.buildDate=$(BUILD_DATE)"
	go mod tidy

serve:
	command -v entr
	find policy -type f | entr -rs 'make gromit && CREDENTIALS='\''{"user":"pass"}'\'' ./gromit policy serve'

test: 
	go test -coverprofile cp.out ./... # dlv test ./cmd #

test-github: test 
	@echo Creates and closes a PR in tyklabs/git-tests
	@GITHUB_TOKEN=$(GITHUB_TOKEN) go test ./policy -run TestGitFunctions

test-jira: test 
	@JIRA_USER=$(JIRA_USER) JIRA_TOKEN=$(JIRA_TOKEN) go test ./policy -run TestJira

update-test-cases:
	@echo Updating test cases for cmd test
	go test ./cmd/ -update

update-actions-versions: bin/update-actions-versions.sed
	echo $(TEMPLATES) | xargs sed -i -f $<

update-variation: policy/templates/test-square/.github/workflows/test-square.yml policy/templates/releng/.github/workflows/release.yml
	sed -i 's/VARIATION: .*/VARIATION: $(VARIATION)/' $^

push: dist/gromit_linux_amd64_v1/gromit
	goreleaser --clean --snapshot
	docker push tykio/gromit:latest

deploy: push
	aws --no-cli-pager ecs update-service --service tui --cluster internal --force-new-deployment
	aws ecs wait services-stable --service tui --cluster internal
	./gromit env expose --env=internal

clean:
	find . -name rice-box.go -o -name error.yaml | xargs rm -fv
	rm -rf $(REPOS)
	rm -fv gromit

unstable-cleanup: gromit
	@PACKAGECLOUD_TOKEN=$(PC_TOKEN) ./gromit pkgs clean --delete $(UNSTABLE_REPOS)

stable-cleanup: gromit
	@PACKAGECLOUD_TOKEN=$(PC_TOKEN) ./gromit pkgs clean $(STABLE_REPOS)

stable-cleanup-really: gromit
	@PACKAGECLOUD_TOKEN=$(PC_TOKEN) ./gromit pkgs clean --delete $(STABLE_REPOS)

sync: gromit
	@$(foreach r,$(REPOS), GITHUB_TOKEN=$(GITHUB_TOKEN) ./gromit policy sync $(r);)

cpr: gromit
	test -n "$(TICKET)"
	@GITHUB_TOKEN=$(GITHUB_TOKEN) JIRA_USER=$(JIRA_USER) JIRA_TOKEN=$(JIRA_TOKEN) ./gromit prs $@ --jira $(TICKET) $(if ifdef REVIEWERS,--reviewers $(REVIEWERS),) $(REPOS)

upr: gromit
	@GITHUB_TOKEN=$(GITHUB_TOKEN) ./gromit prs $@ $(REPOS)

opr: gromit
	@GITHUB_TOKEN=$(GITHUB_TOKEN) ./gromit prs $@ $(REPOS)

loc: clean
	gocloc --skip-duplicated --not-match-d=\.terraform --output-type=json ~gromit ~ci | jq -r '.languages | map([.name, .code]) | transpose[] | @csv'

.PHONY: clean update-test-cases test loc cpr upr opr deploy push

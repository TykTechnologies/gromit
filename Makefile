SHELL := bash
VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)

# A directory containing config files
CONF_VOL := testdata
UNAME_S := $(shell uname -s)

gromit: */*.go confgen/templates/* policy/templates/* #devenv/terraform/* server/debug/debugger.wasm
	! ls **/#*#
	go build -v -trimpath -ldflags "-X github.com/TykTechnologies/gromit/util.version=$(VERSION) -X github.com/TykTechnologies/gromit/util.commit=$(COMMIT) -X github.com/TykTechnologies/gromit/util.buildDate=$(BUILD_DATE)"
	go mod tidy
ifneq ($(UNAME_S),Darwin)
	sudo setcap 'cap_net_bind_service=+ep' $(@)
endif

testdata: testdata/base/*
	[[ $CI ]] || test -n "$(AWS_PROFILE)"
	cd testdata/base
	terraform init
	terraform apply -auto-approve

test: 
	echo Use a config file locally and env variables in CI
	go test -coverprofile cp.out ./... # dlv test ./cmd #

licenser: clean
	docker build -t $(@) . && docker run --rm --name $(@) \
	-e LICENSER_TOKEN=$(token) \
	--mount type=bind,src=$(PWD)/$(CONF_VOL),target=/config \
	$(@) licenser dashboard-trial /config/dash.test

server/debug/debugger.wasm: */*.go
	GOOS=js GOARCH=wasm go build -o $(@)

clean:
	find . -name rice-box.go | xargs rm -fv
	rm -fv gromit

.PHONY: grun gserve clean

SHELL := bash
VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)

# A docker volume, can be empty for testing, will have data in it after
CONF_VOL := testdata

gromit: */*.go
	go generate ./terraform ./confgen
	go build -v -trimpath -ldflags "-X 'github.com/TykTechnologies/gromit/util.version=$(VERSION)' -X 'github.com/TykTechnologies/gromit/util.commit=$(COMMIT)' -X 'github.com/TykTechnologies/gromit/util.buildDate=$(BUILD_DATE)'"
	go mod tidy
#	sudo setcap 'cap_net_bind_service=+ep' $(@)

testdata: testdata/base/*
	[[ $CI ]] || test -n "$(AWS_PROFILE)"
	cd testdata/base
	terraform init
	terraform apply -auto-approve

test: 
	echo Use a config file locally and env variables in CI
	go test ./... # dlv test ./cmd #

grun: clean
	docker build -t $(@) . && docker run --rm --name $(@) \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	-e AWS_ACCESS_KEY_ID=$(aws_id) \
	-e AWS_SECRET_ACCESS_KEY=$(aws_key) \
	-e AWS_REGION=eu-central-1 \
	-e TF_API_TOKEN=$(tf_api) \
	-e GROMIT_DOMAIN=dev.tyk.technology \
	-e GROMIT_ZONEID=Z06422931MJIQS870BBM7 \
	--mount type=bind,src=$(PWD)/$(CONF_VOL),target=/config \
	grun -l trace cluster run /config

licenser: clean
	docker build -t $(@) . && docker run --rm --name $(@) \
	-e LICENSER_TOKEN=$(token) \
	--mount type=bind,src=$(PWD)/$(CONF_VOL),target=/config \
	$(@) licenser dashboard-trial /config/dash.test

gserve: clean
	docker build -t gserve . && docker run --rm --name $(@) \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	grun serve

clean:
	find . -name rice-box.go | xargs rm -fv
	rm -fv gromit

.PHONY: grun gserve clean

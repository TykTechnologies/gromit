VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)

gromit: */*.go
	go build -trimpath -ldflags "-X util.Version=$(VERSION) -X util.Commit=$(COMMIT) -X util.BuildDate=$(BUILD_DATE)"
	rice embed-go
	go mod tidy
	#sudo setcap 'cap_net_bind_service=+ep' $(@)

grun: gromit
	docker build -t grun . && docker run --rm --name $(@) \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	-e AWS_ACCESS_KEY_ID=$(aws_id) \
	-e AWS_SECRET_ACCESS_KEY=$(aws_key) \
	-e AWS_REGION=eu-central-1 \
	-e TF_API_TOKEN=$(tf_api) \
	-e GROMIT_DOMAIN=dev.tyk.technology \
	-e GROMIT_ZONEID=Z06422931MJIQS870BBM7 \
	grun run

gserve: gromit
	docker build -t gserve . && docker run --rm --name $(@) \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	grun run

.PHONY: grun

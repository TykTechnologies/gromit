VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-list -1 HEAD)
BUILD_DATE := $(shell date +%FT%T%z)

gromit: */*.go
	find . -name rice_box.go | xargs rm -fv
	rice -v embed-go -i ./terraform -i ./confgen
	go build -trimpath -ldflags "-X util.Version=$(VERSION) -X util.Commit=$(COMMIT) -X util.BuildDate=$(BUILD_DATE)"
	go mod tidy
	#sudo setcap 'cap_net_bind_service=+ep' $(@)

grun: clean
	docker build -t grun . && docker run --rm --name $(@) \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	-e AWS_ACCESS_KEY_ID=$(aws_id) \
	-e AWS_SECRET_ACCESS_KEY=$(aws_key) \
	-e AWS_REGION=eu-central-1 \
	-e TF_API_TOKEN=$(tf_api) \
	-e GROMIT_DOMAIN=dev.tyk.technology \
	-e GROMIT_ZONEID=Z06422931MJIQS870BBM7 \
	grun cluster run /config

gserve: clean
	docker build -t gserve . && docker run --rm --name $(@) \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	grun serve

clean:
	rm -fv gromit

.PHONY: grun gserve clean

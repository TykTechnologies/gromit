gromit: *.go cmd/*.go devenv/*.go terraform/*.go server/*.go
	go build
	cd terraform && rice append --exec ../$(@)
	go mod tidy
	sudo setcap 'cap_net_bind_service=+ep' $(@)

grun: gromit
	docker build -t grun . && docker run --rm --name gr0 \
	-e GROMIT_TABLENAME=DeveloperEnvironments \
	-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
	-e AWS_ACCESS_KEY_ID=$(aws_id) \
	-e AWS_SECRET_ACCESS_KEY=$(aws_key) \
	-e AWS_REGION=eu-central-1 \
	-e TF_API_TOKEN=$(tf_api) \
	-e GROMIT_DOMAIN=dev.tyk.technology \
	grun run

.PHONY: grun

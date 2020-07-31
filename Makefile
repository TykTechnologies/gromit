gromit: *.go cmd/*.go devenv/*.go terraform/*.go server/*.go
	rice embed-go
	go build
	go mod tidy
	sudo setcap 'cap_net_bind_service=+ep' $(@)

serve: gromit
	GROMIT_TABLENAME=DeveloperEnvironments GROMIT_REPOS=tyk,tyk-analytics,tyk-pump GROMIT_REGISTRYID=046805072452 ./gromit serve --certpath scerts


serve: gromit
	GROMIT_TABLENAME=DeveloperEnvironments GROMIT_REPOS=tyk,tyk-analytics,tyk-pump GROMIT_REGISTRYID=046805072452 ./gromit serve --certpath scerts

gromit: *.go cmd/*.go devenv/*.go
	go build
	go mod tidy
	sudo setcap 'cap_net_bind_service=+ep' $(@)

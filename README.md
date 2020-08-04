# Gromit

This is ostensibly a [cobra](https://github.com/spf13/cobra "cobra cli") app. 

``` shellsession
% ./gromit --help
A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.

Usage:
  gromit [command]

Available Commands:
  client      Interact with the gromit server
  help        Help about any command
  serve       Run endpoint for github requests

Flags:
      --config string   config file (default is $HOME/.gromit.yaml)
  -h, --help            help for gromit
  -t, --toggle          Help message for toggle

Use "gromit [command] --help" for more information about a command.
```

## Server

This runs at `gromit.dev.tyk.technology` and listens for the requests that come in from the [int-images](https://github.com/TykTechnologies/tyk-ci/blob/master/wf-gen/int-image.yml.m4 "integration images") Github Actions.

```shellsession
% ./gromit serve --help
Runs an HTTPS server, bound to 443 that can be accesses only via mTLS. 

This endpoint is notified by the int-image workflows in the various repos when there is a new build

Usage:
  gromit serve [flags]

Flags:
      --certpath string   path to rootca and key pair. Expects files named ca.pem, server(-key).pem (default "certs")
  -h, --help              help for serve

Global Flags:
      --config string   config file (default is $HOME/.gromit.yaml)
```

# To test gromit run locally

```
docker build -t grun . && docker run --rm --name gr0 \
-e GROMIT_TABLENAME=DeveloperEnvironments \
-e GROMIT_REPOS=tyk,tyk-analytics,tyk-pump \
-e AWS_ACCESS_KEY_ID=keyid \
-e AWS_SECRET_ACCESS_KEY=secret \
-e AWS_REGION=eu-central-1 \
-e TF_API_TOKEN=secret \
-e GROMIT_DOMAIN=dev.tyk.technology \
grun run
```

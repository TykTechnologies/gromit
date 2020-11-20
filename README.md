![Docker build](https://github.com/TykTechnologies/gromit/workflows/Release/badge.svg?branch=master)

# Gromit

This is ostensibly a [cobra](https://github.com/spf13/cobra "cobra cli") app. 

``` shellsession
The subcommands run as services and scheduled tasks in the internal cluster.
Global env vars:
These vars apply to all commands
GROMIT_TABLENAME DynamoDB tablename to use for env state
GROMIT_REPOS Comma separated list of ECR repos to answer for

Usage:
  gromit [command]

Available Commands:
  client      Interact with the gromit server
  expose      Upsert a record in Route53 for the given ECS cluster
  help        Help about any command
  redis       Dump redis keys to files
  run         Process envs from GROMIT_TABLENAME
  serve       Run endpoint for github requests

Flags:
      --gconf string   config file (default is $HOME/.gromit.yaml)
  -h, --help           help for gromit
  -t, --toggle         Help message for toggle

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

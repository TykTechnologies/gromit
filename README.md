![Release](https://github.com/TykTechnologies/gromit/workflows/Release/badge.svg?branch=master)

# Gromit

## Installation
Install from the [releases](releases) page. To keep up with releases using [zinit](https://github.com/zdharma/zinit) in turbo mode, 

``` shell
zinit wait lucid from"gh-r" nocompile for \
      bpick"*Linux_x86_64.tar.gz" TykTechnologies/gromit
```
## Configuration
This is ostensibly a [cobra](https://github.com/spf13/cobra "cobra cli") app and can be configured with a config file to save a bunch of typing. A sample `gromit.yaml` file looks like:

``` yaml
env:
  ccerts:
    ca: ccerts/ca.pem
    key: ccerts/key.pem
    cert: ccerts/cert.pem
  authtoken: supersekret

```
## Features
``` shellsession
% gromit help
It also has a grab bag of various ops automation.
Global env vars:
These vars apply to all commands
GROMIT_TABLENAME DynamoDB tablename to use for env state
GROMIT_REPOS Comma separated list of ECR repos to answer for

Usage:
  gromit [command]

Available Commands:
  cluster     Manage cluster of tyk components
  env         Mess about with the env state
  help        Help about any command
  licenser    Get a trial license and writes it to path, overwriting it.
  orgs        Dump/restore org keys and mongodb
  serve       Run endpoint for github requests
  version     Print version

Flags:
      --conf string       config file (default is $HOME/.config/gromit.yaml)
  -h, --help              help for gromit
  -l, --loglevel string   Log verbosity: trace, info, warn, error (default "info")

Use "gromit [command] --help" for more information about a command.
```

## Testing
Only system tests exist and these will exercise most of the AWS API code. `make test` runs the tests and requires access to the [Engg PoC](https://046805072452/signing/aws/amazon.com/console/) AWS account.

The tests depend on ECR repos being present. These repos are also used by the CI workflow whose badge is at the top of this README. If you need to create the repos for whatever reason, you can do so by running terraform in <testdata/base>. 

If your AWS account does not have the power to run the tests, please post in #devops.

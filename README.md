![Release](https://github.com/TykTechnologies/gromit/workflows/Release/badge.svg?branch=master)

# Gromit

## Installation
Install from the [releases](releases) page. To keep up with releases using [zinit](https://github.com/zdharma/zinit) in turbo mode, 

``` shell
zinit wait lucid from"gh-r" nocompile for \
      bpick"*Linux_x86_64.tar.gz" TykTechnologies/gromit
```

## Configuration
This is ostensibly a [cobra](https://github.com/spf13/cobra "cobra cli") app and can be configured with a config file to save a bunch of typing. The embedded [config.yaml](config/config.yaml) contains the configuration that drives the templates.

All parameters can also be set by environment variables with the `GROMIT_` prefix. So the environment variable for the config parameter `cluster.domain` would be `GROMIT_CLUSTER_DOMAIN`.

## Features
To various degrees of competence, gromit can,
- manage templated files that can be rendered into any repo under management
  * releng
  * gpac (github policy as code)
- fetch developer licenses for dashboard and mdcb
- generate config files from a `text/template`
- dump redis and mongo data for a classic cloud org to local disk (broken)
- restore redis and mongo data for a classic cloud org from local disk (broken)

### Policy Engine for release engineering
Policies are implemented by rendering template bundles, which are usually embedded into the binary. The rendering is mere text substitution and is agnostic to the language used in the template. It is best to use declarative or some sort of well-understood configuration language like YAML in the templates though.

#### releng
This bundle contains all of the code required to build and test all the artefacts that are created when a release is made. Releases are made by pushing a tag to github. 

#### gpac
This bundle implements terraform manifests that model the state of the repos under management in github. This is used to [keep track of release branches](https://tyktech.atlassian.net/wiki/spaces/EN/pages/1907228677/Release+branches) as they are created.


### Usage
``` shellsession
% gromit help
It also has a grab bag of various ops automation.
Each gromit command has its own config section. For instance, the policy command uses the policy key in the config file. Config values can be overridden by environment variables. For instance, policy.prefix can be overridden using the variable $GROMIT_POLICY_PREFIX.

Usage:
  gromit [command]

Available Commands:
  bundle      Operate on bundles
  completion  Generate completion script
  git         Top-level git command, use a sub-command to perform an operation
  help        Help about any command
  licenser    Get a trial license and writes it to path, overwriting it
  orgs        Dump/restore org keys and mongodb
  passwd      Returns the password hash of the given plaintext
  policy      Templatised policies that are driven by the config file
  version     Print version

Flags:
  -f, --conf string       YAML config file. If not supplied, embedded defaults will be used
  -h, --help              help for gromit
      --loglevel string   Log verbosity: trace, debug, info, warn, error/fatal (default "info")
      --textlogs          Logs in plain text (default true)

Use "gromit [command] --help" for more information about a command.
```

## Testing
All tests in the `cmd` directory are system tests. Tests in other directories are unit tests. `make test` runs the tests and requires access to the [Engg PoC](https://046805072452/signing/aws/amazon.com/console/) AWS account.

If your AWS account does not have the power to run the tests, please find us on Slack.

## CI
The [Release](https://github.com/TykTechnologies/gromit/actions?query=workflow%3ARelease) action builds a new docker image.

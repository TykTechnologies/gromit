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
authtoken: supersekret
tablename: GromitTest
registryid: "046805072452"
repos: "tyk,tyk-analytics,tyk-pump"

cluster:
  zoneid: Z02045551IU0LZIOX4AO0
  domain: test.tyk.technology

licenser:
  bot: <license_server_url>
  dash:
    token: supersekret
    api: dashboard-trial
  mdcb:
    token: supersekret
    api: mdcb-trial?auth=supersekret

ca: |
  <paste PEM>
  
serve:
  key : |
    <paste PEM>
    
  cert: |
    <paste PEM>

client:
  key: |
	<paste PEM>
  
  cert: |
	<paste PEM>
```

All parameters can also be set by environment variables with the `GROMIT_` prefix. So the environment variable for the config parameter `cluster.domain` would be `GROMIT_CLUSTER_DOMAIN`.

## Features
To various degrees of competence, gromit can,
- wait for new builds from the Release workflow in repos and persist the current state to DB
- read build state from DB and update the developer environments with latest images
- manage the meta-automation for the release process ([sync-automation.yml](policy/templates/sync-automation.tmpl "template"))
- fetch developer licenses for dashboard and mdcb
- generate config files from a `text/template`
- dump redis and mongo data for a classic cloud org to local disk
- restore redis and mongo data for a classic cloud org from local disk

### Policy Engine for release engineering
If it is told (via the config file), gromit can manage the forward and back porting of the release engineering code. [RFC](https://tyktech.atlassian.net/wiki/spaces/EN/pages/1030586370/Keeping+release+engineering+code+in+sync) here. Given a policy definition like
``` yaml
policy:
  protected: [ branches_that_are_protected on_github ]
  files:
    - file1
    - .goreleaser.yml
    - Dockerfile.std
  repos:
    tyk:
      deprecations:
        <version_when_deprecated>:
          - file_that_was_deprecated
          - bin/integration_build.sh
      backports:
        release-3.0.5: releng/release-3-lts
        <source_branch>: <backport_branch>
    repo2:
      files:
        - .github/workflows/update-gomod.yml
        - .github/workflows/build-assets.yml
      deprecations:
        v3.0.1:
          - .github/workflows/int-image.yml
          - bin/integration_build.sh
      backports:
        release-3.0.5: releng/release-3-lts
        release-3.1.2: releng/release-3.1
```
gromit will generate a `.g/w/sync-automation.yml` file in each `<source_branch>` which will copy all files related to release engineering to `<backport_branch>`. The `<backport_branch>` can be merged into its ancestor branch at periodic intervals. 

For the example `tyk` repo above, commits on `release-3.0.5` related to release engineering will be copied to `releng/release-3-lts`. `releng/release-3-lts` can be merged, via a PR manually, or auotmatically, into `release-3-lts` as part of the release process.

### Usage
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
  licenser    Get a trial license and writes it to path, overwriting it
  orgs        Dump/restore org keys and mongodb
  passwd      Returns the password hash of the given plaintext
  policy      Mess with the release engineering policy
  reap        Reap envs from GROMIT_TABLENAME, using a config tree at <config root path>
  repo        Work with git repos
  serve       Run endpoint for github requests
  sow         Sow envs creating a config tree at <config root path>
  version     Print version

Flags:
  -f, --conf string       config file (default is $HOME/.config/gromit.yaml)
  -h, --help              help for gromit
  -l, --loglevel string   Log verbosity: trace, info, warn, error (default "info")
  -t, --textlogs          Logs in plain text

Use "gromit [command] --help" for more information about a command.
```

## Testing
All tests in the `cmd` directory are system tests. Tests in other directories are unit tests. `make test` runs the tests and requires access to the [Engg PoC](https://046805072452/signing/aws/amazon.com/console/) AWS account.

The tests depend on:
- ECR repos
- DynamoDB table
- some other AWS stuff, see config file

This infra is provisioned in the Engg PoC account and can be found in the `devenv-euc1-test` Terraform workspace for the state in [tyk-ci/infra](https://github.com/TykTechnologies/tyk-ci/tree/master/infra).

If your AWS account does not have the power to run the tests, please post in #devops.

## CD
The [Release](https://github.com/TykTechnologies/gromit/actions?query=workflow%3ARelease) action builds a new docker image and notifies tyk-ci about the new version. Actions on tyk-ci implement further automation.

## Certificates
Import the cfssl provided certificates into your local trust hierarchy so that you don't have to futz about with command line args for curl and so on.

### ca-certificates
Copy `rootca.pem` to `/usr/share/ca-certificates/gromit/rootca.crt`, creating the directory if it does not exist.
Add `gromit/rootca.crt` to `/etc/ca-certificates.conf`.
Run `sudo dpkg-reconfigure ca-certificates`.

### Chrome
It looks like Chrome doesn't trust the local ca-certificates. You can add it to the per-user nss store in `~/.pki/nssdb` as per [per the docs](https://chromium.googlesource.com/chromium/src/+/master/docs/linux/cert_management.md).

``` shellsession
% apt install libnss3-tools
% certutil -d .pki/nssdb -A -t "C,," -n gromit -i /usr/share/ca-certificates/gromit/rootca.crt
```

Import your client certificates with after converting it to PKCS#12/PFX form,

``` shellsession
% openssl pkcs12 -export -out gclient.p12 -inkey ~gromit/testdata/gromit/ccerts/key.pem -in ~gromit/testdata/gromit/ccerts/cert.pem -certfile ~ci/certs/rootca/rootca.pem 
% pk12util -d sql:.pki/nssdb -i gclient.p12 -n gromitclient
```

### curl

``` shellsession
% curl -v --key ~gromit/testdata/gromit/ccerts/key.pem --cert ~gromit/testdata/gromit/ccerts/cert.pem https://127.0.0.1/healthcheck
*   Trying 127.0.0.1:443...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*   CAfile: /etc/ssl/certs/ca-certificates.crt
  CApath: /etc/ssl/certs
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
* TLSv1.3 (IN), TLS handshake, Request CERT (13):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
* TLSv1.3 (IN), TLS handshake, Finished (20):
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.3 (OUT), TLS handshake, Certificate (11):
* TLSv1.3 (OUT), TLS handshake, CERT verify (15):
* TLSv1.3 (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384
* ALPN, server accepted to use http/1.1
* Server certificate:
*  subject: C=UK; ST=Greater London; L=London; O=Tyk Technologies; OU=Devops; CN=Test Cert
*  start date: Apr 19 08:35:00 2021 GMT
*  expire date: Apr 19 08:35:00 2022 GMT
*  subjectAltName: host "127.0.0.1" matched cert's IP address!
*  issuer: C=UK; ST=Greater London; L=London; O=Tyk Technologies; OU=Devops; CN=Tyk Developer Environments
*  SSL certificate verify ok.
> GET /healthcheck HTTP/1.1
> Host: 127.0.0.1
> User-Agent: curl/7.68.0
> Accept: */*
> 
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Date: Sun, 25 Apr 2021 08:48:37 GMT
< Content-Length: 2
< Content-Type: text/plain; charset=utf-8
< 
* Connection #0 to host 127.0.0.1 left intact
OK%
```

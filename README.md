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
  reap        Reap envs from GROMIT_TABLENAME, using a config tree at <config root path>
  serve       Run endpoint for github requests
  sow         Sow envs creating a config tree at <config root path>
  version     Print version

Flags:
      --conf string       config file (default is $HOME/.config/gromit.yaml)
  -h, --help              help for gromit
  -l, --loglevel string   Log verbosity: trace, info, warn, error (default "info")
  -t, --textlogs          Logs in plain text

Use "gromit [command] --help" for more information about a command.
```

## Testing
Only system tests exist and these will exercise most of the AWS API code. `make test` runs the tests and requires access to the [Engg PoC](https://046805072452/signing/aws/amazon.com/console/) AWS account.

The tests depend on:
- ECR repos
- DynamoDB table
- some other AWS stuff

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

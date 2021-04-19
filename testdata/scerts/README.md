# Testing server certificates
These are special certs for localhost and 127.0.0.1. Use these in the automated test suite. They are signed by the same CA as for production.

When creating via cfssl, the command is,
```
CFSSL_API_KEY=sekret cfssl gencert -config ~src/cfssl/remote-sign.json -profile=server -hostname=127.0.0.1,127.0.0.2,127.0.0.3 csr.json | cfssljson -bare s
```

FROM golang:1.14 as builder 

RUN CGO_ENABLED=0 go get github.com/TykTechnologies/gromit

# generate clean, final image for end users
FROM alpine:latest
COPY --from=builder /go/bin/* /usr/bin/

EXPOSE 443
VOLUME /cfssl
WORKDIR /cfssl

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve", "--certpath=gromit/server" ]

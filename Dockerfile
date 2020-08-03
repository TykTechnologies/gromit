FROM golang:1.14 as builder 

RUN go get -u github.com/GeertJohan/go.rice/rice
WORKDIR /src/gromit
ADD . .
RUN CGO_ENABLED=0 go build && rice append --exec gromit

# generate clean image for end users
FROM alpine:latest
COPY --from=builder /src/gromit/gromit /usr/bin/

EXPOSE 443
VOLUME /cfssl
WORKDIR /cfssl

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve", "--certpath=gromit/server" ]

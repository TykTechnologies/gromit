FROM golang:1.14 as builder 

RUN apt-get update && apt-get install -y unzip && go get -u github.com/GeertJohan/go.rice/rice
WORKDIR /src/gromit
RUN curl https://releases.hashicorp.com/terraform/0.13.0-rc1/terraform_0.13.0-rc1_linux_amd64.zip -o terraform.zip && unzip terraform.zip && mv terraform /
ADD . .
RUN CGO_ENABLED=0 go build && rice embed-go

# generate clean image for end users
FROM alpine:latest
COPY --from=builder /src/gromit/gromit /usr/bin/
COPY --from=builder /terraform /usr/bin/

EXPOSE 443
VOLUME /cfssl
WORKDIR /cfssl

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve", "--certpath=gromit/server" ]

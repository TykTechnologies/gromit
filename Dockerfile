FROM golang:1.14 as builder 

ARG TF_VER=0.13.4

RUN apt-get update && apt-get install -y unzip && go get -u github.com/GeertJohan/go.rice/rice
WORKDIR /src/gromit
RUN curl https://releases.hashicorp.com/terraform/${TF_VER}/terraform_${TF_VER}_linux_amd64.zip -o terraform.zip && unzip terraform.zip && mv terraform /
ADD . .
RUN CGO_ENABLED=0 make gromit

# generate clean image for end users
FROM alpine:latest
COPY --from=builder /src/gromit/gromit /usr/bin/
COPY --from=builder /terraform /usr/bin/
EXPOSE 443
RUN mkdir /config /cfssl
VOLUME [ "/cfssl" "/config" ]

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve", "--certpath=gromit/server" ]

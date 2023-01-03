FROM golang:1.18 as builder 

RUN apt-get update && apt-get install -y unzip
WORKDIR /src/gromit
ADD . .
RUN CGO_ENABLED=0 make gromit

# generate clean image for end users
FROM alpine:latest
RUN apk update && apt add git
COPY --from=builder /src/gromit/gromit /usr/bin/
EXPOSE 443
RUN mkdir /config /cfssl
VOLUME [ "/cfssl" "/config" ]

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve" ]

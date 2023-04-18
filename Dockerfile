FROM golang:1.20 as builder 

RUN apt-get update && apt-get install -y unzip
WORKDIR /src/gromit
ADD . .
RUN CGO_ENABLED=0 make gromit

# generate clean image for end users
FROM debian:stable-slim
RUN apt-get update && apt-get dist-upgrade -y git curl
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install gh -y
COPY --from=builder /src/gromit/gromit /usr/bin/
EXPOSE 443
RUN mkdir /config
VOLUME [ "/config" ]

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve" ]

FROM debian:stable-slim@sha256:99fc6d2a0882fcbcdc452948d2d54eab91faafc7db037df82425edcdcf950e1f
RUN apt-get update && apt-get dist-upgrade -y git curl
# TODO(security): curl piped to shell - consider pre-downloading and verifying checksum
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install gh -y
COPY gromit /usr/bin/
EXPOSE 443

# executable
ENTRYPOINT [ "gromit" ]
# arguments that can be overridden
CMD [ "serve" ]

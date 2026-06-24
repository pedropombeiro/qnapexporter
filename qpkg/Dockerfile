FROM ubuntu:20.04@sha256:8feb4d8ca5354def3d8fce243717141ce31e2c428701f6682bd2fafe15388214

ARG QDK_VER=2.5.3

# install curl (ca-certificates is required for TLS and is otherwise pulled in
# as a recommended package, which --no-install-recommends excludes)
# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

# download QDK
RUN mkdir qdk \
    && curl -L -o qdk/v${QDK_VER}.tar.gz https://github.com/qnap-dev/QDK/archive/refs/tags/v${QDK_VER}.tar.gz

# install QDK
#
# InstallToUbuntu.sh compiles qpkg_encrypt and installs it to /bin (on PATH),
# where qbuild finds it as a bare command. qbuild's encryption step populates
# the qpkg trailer checksum; if qpkg_encrypt were missing the checksum would be
# left blank and QTS would reject the package with a "file format error" (see
# https://github.com/pedropombeiro/qnapexporter/issues/103, which affected QDK
# 2.5.0 where the /bin copy was dropped; it was restored in 2.5.1). The
# validate-tail.sh build/release check guards against a regression.
# hadolint ignore=DL3003
RUN cd qdk \
    && tar xvfz v${QDK_VER}.tar.gz \
    && cd QDK-${QDK_VER} \
    && ./InstallToUbuntu.sh install \
    && rm -rf /var/lib/apt/lists/*

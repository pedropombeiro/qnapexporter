FROM ubuntu:20.04@sha256:8feb4d8ca5354def3d8fce243717141ce31e2c428701f6682bd2fafe15388214

ARG QDK_VER=2.5.2

# install curl
# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

# download QDK
RUN mkdir qdk \
    && curl -L -o qdk/v${QDK_VER}.tar.gz https://github.com/qnap-dev/QDK/archive/refs/tags/v${QDK_VER}.tar.gz

# install QDK
#
# As of QDK 2.5.0, InstallToUbuntu.sh compiles qpkg_encrypt into src/bin/ but
# only copies shared/ to /usr/share/QDK, so qpkg_encrypt is missing from PATH.
# qbuild invokes qpkg_encrypt as a bare command (relying on PATH); without it
# the encryption step silently no-ops and leaves the qpkg trailer checksum
# field blank, which QTS rejects with a "file format error"
# (see https://github.com/pedropombeiro/qnapexporter/issues/103). Install the
# compiled binary into /usr/local/bin so it is always on PATH (QDK 2.4.0
# installed it to /bin for the same reason).
# hadolint ignore=DL3003
RUN cd qdk \
    && tar xvfz v${QDK_VER}.tar.gz \
    && cd QDK-${QDK_VER} \
    && ./InstallToUbuntu.sh install \
    && cp src/bin/qpkg_encrypt /usr/local/bin/qpkg_encrypt \
    && rm -rf /var/lib/apt/lists/*

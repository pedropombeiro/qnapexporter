FROM ubuntu:24.04@sha256:c35e29c9450151419d9448b0fd75374fec4fff364a27f176fb458d472dfc9e54

ARG QDK_VER=2.3.14

# install wget
RUN apt-get update \
    && apt-get install -y curl \
    && rm -rf /var/lib/apt/lists/*

# download QDK
RUN mkdir qdk \
    && curl -L -o qdk/v${QDK_VER}.tar.gz https://github.com/qnap-dev/QDK/archive/refs/tags/v${QDK_VER}.tar.gz

# install QDK
RUN cd qdk \
    && tar xvfz v${QDK_VER}.tar.gz \
    && cd QDK-${QDK_VER} \
    && ./InstallToUbuntu.sh install \
    && rm -rf /var/lib/apt/lists/*

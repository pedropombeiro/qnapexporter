# qnapexporter

`qnapexporter` is a simple Go program meant to be run in the background on a QNAP NAS in order to periodically export
relevant metrics to Prometheus. It generates a file on the local filesystem which can be exposed to Prometheus using a
web server such as Nginx.

The data produced by this exporter can be used to create a Grafana dashboard such as the following:

![Grafana dashboard sample](assets/grafana.png "Grafana dashboard sample")

## Installation

1. Assuming you have Entware installed on your NAS, install Go:

    ```shell
    opkg install go
    export GOROOT=/opt/bin/go
    export PATH="${PATH}:${GOROOT}/bin"
    ```

1. Install `qnapexporter`

    ```shell
    go get -u gitlab.com/pedropombeiro/qnapexporter
    ```

1. Run `qnapexporter`

    ```shell
    ~/go/bin/qnapexporter
    ```

    Normally it should be run as a background task. Unfortunately this is not easy on a QNAP NAS.
    See for example [this forum post](https://forum.qnap.com/viewtopic.php?t=44743#p198192) for ideas on how to achieve it.

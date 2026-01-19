#!/bin/sh

CONF=/etc/config/qpkg.conf
QPKG_NAME=QNAPExporter
QPKG_DIR=$(getcfg $QPKG_NAME Install_Path -f $CONF)
PID_FILE=/var/run/qnapexporter.pid

# see https://github.com/pedropombeiro/qnapexporter for customization options
EXTRA_ARGS=""

# Source environment file if it exists (for auth tokens and other config)
ENV_FILE="$QPKG_DIR/.env"
if [ -f "$ENV_FILE" ]; then
    set -a
    . "$ENV_FILE"
    set +a
fi

case "$1" in
  start)
    ENABLED=$(getcfg $QPKG_NAME Enable -u -d FALSE -f $CONF)
    if [ "$ENABLED" != "TRUE" ]; then
        echo "$QPKG_NAME is disabled."
        exit 1
    fi

    if [ -f $PID_FILE ] && kill -0 $(cat $PID_FILE); then
      echo "$QPKG_NAME is already running."
      exit 1
    else
      $QPKG_DIR/qnapexporter $EXTRA_ARGS &
      echo $! > $PID_FILE
    fi
    ;;

  stop)
    if [ -f $PID_FILE ]; then
      PID=$(cat $PID_FILE)
      if kill -0 $PID; then
        kill $PID
        while [ -e /proc/$PID ]; do
          sleep 1;
        done
      fi
      rm $PID_FILE
    else
      echo "$QPKG_NAME is not running."
      exit 1
    fi
    ;;

  restart)
    $0 stop
    $0 start
    ;;

  *)
    echo "Usage: $0 {start|stop|restart}"
    exit 1
esac

exit 0

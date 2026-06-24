#!/bin/sh

CONF=/etc/config/qpkg.conf
QPKG_NAME=QNAPExporter
QPKG_DIR=$(getcfg $QPKG_NAME Install_Path -f $CONF)
PID_FILE=/var/run/qnapexporter.pid

# Seconds to wait before relaunching the binary after an unexpected exit.
RESTART_BACKOFF=5

# see https://github.com/pedropombeiro/qnapexporter for customization options
EXTRA_ARGS=""

# Source environment file if it exists (for auth tokens and other config)
ENV_FILE="$QPKG_DIR/.env"
if [ -f "$ENV_FILE" ]; then
  set -a
  # shellcheck source=/dev/null
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

  if [ -f $PID_FILE ] && kill -0 "$(cat $PID_FILE)" 2>/dev/null; then
    echo "$QPKG_NAME is already running."
    exit 1
  else
    # Launch a supervisor that keeps the binary running. If the binary exits
    # unexpectedly, the supervisor relaunches it after a short backoff. The
    # supervisor traps TERM so that `stop` cleanly kills both the loop and the
    # running child, without the child being respawned.
    (
      CHILD_PID=""
      trap 'if [ -n "$CHILD_PID" ]; then kill "$CHILD_PID" 2>/dev/null; fi; exit 0' TERM INT
      while true; do
        # EXTRA_ARGS is intentionally word-split to pass multiple flags.
        # shellcheck disable=SC2086
        "$QPKG_DIR"/qnapexporter $EXTRA_ARGS &
        CHILD_PID=$!
        wait "$CHILD_PID"
        CHILD_PID=""
        sleep "$RESTART_BACKOFF"
      done
    ) &
    echo $! >$PID_FILE
  fi
  ;;

stop)
  if [ -f $PID_FILE ]; then
    PID=$(cat $PID_FILE)
    if kill -0 "$PID" 2>/dev/null; then
      # Terminate the supervisor; its TERM trap kills the running binary and
      # stops the restart loop.
      kill "$PID" 2>/dev/null
      while kill -0 "$PID" 2>/dev/null; do
        sleep 1
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
  ;;
esac

exit 0

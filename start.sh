#!/bin/sh
# ARTEX   (Linux / macOS / Docker ENTRYPOINT)
#
#   ./start.sh                        (Ctrl-C  )
#   nohup ./start.sh >artex.log 2>&1 &    
#   ./start.sh -addr :9000             artex   
#
#
#
#  ,        .  ·  Go(selfupdate ) 
set -u

cd "$(dirname "$0")" || exit 1

BIN=./artex
[ -x "$BIN" ] || { echo "[artex]     : $BIN" >&2; exit 1; }

RESTART_CODE=75
MAX_DELAY=60

child=0
stopping=0

#
# Docker   . docker stop  SIGTERM  PID 1(  ) 
forward() {
	stopping=1
	if [ "$child" -ne 0 ]; then
		kill -TERM "$child" 2>/dev/null || true
	fi
}
trap forward INT TERM

delay=1
while :; do
	"$BIN" "$@" &
	child=$!

	wait "$child"
	code=$?
	if [ "$code" -gt 128 ]; then
		wait "$child"
		code=$?
	fi
	child=0

	if [ "$stopping" -eq 1 ]; then
		echo "[artex] "
		exit 0
	fi

	case "$code" in
		0)
			echo "[artex]  "
			exit 0
			;;
		"$RESTART_CODE")
			# / .   artex      (selfupdate.Bootstrap ).
			echo "[artex]  (  )…"
			delay=1
			;;
		*)
			echo "[artex]   (code=$code), ${delay}s  " >&2
			sleep "$delay"
			delay=$((delay * 2))
			[ "$delay" -gt "$MAX_DELAY" ] && delay=$MAX_DELAY
			;;
	esac
done

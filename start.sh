#!/bin/sh
# Blacksand supervisor: exit 0 stops, 75 restarts immediately, other codes back off.
# Forward termination signals and wait for the child to finish graceful shutdown.
# Download verification and atomic replacement are handled by selfupdate.Bootstrap.
set -u


cd "$(dirname "$0")" || exit 1

BIN=./blacksand
[ -x "$BIN" ] || { echo "[Blacksand] Executable not found: $BIN" >&2; exit 1; }

RESTART_CODE=75
MAX_DELAY=60

child=0
stopping=0

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
		echo "[Blacksand] Stopped"
		exit 0
	fi

	case "$code" in
		0)
			echo "[Blacksand] Exited normally"
			exit 0
			;;
		"$RESTART_CODE")
			echo "[Blacksand] Restart requested (applying update)…"
			delay=1
			;;
		*)
			echo "[Blacksand] Exited with code $code; restarting in ${delay}s" >&2
			sleep "$delay"
			delay=$((delay * 2))
			[ "$delay" -gt "$MAX_DELAY" ] && delay=$MAX_DELAY
			;;
	esac
done

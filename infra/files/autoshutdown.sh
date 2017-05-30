#!/bin/bash

get_shutdown_idle_seconds() {
    config_value=$(cat /etc/talkative_idle_shutdown_seconds 2>/dev/null || true)
    echo ${config_value:=$((60 * 60 * 12))}
}
POLLING_INTERVAL=15
APP_TO_SEARCH=talkative_s

echo shutdown idle timeout: `get_shutdown_idle_seconds`s


last_alive=$(date +%s)
while true
do
        SHUTDOWN_IDLE_SECONDS=`get_shutdown_idle_seconds`
        pids=$(pgrep "$APP_TO_SEARCH")
        if [ "$?" -eq 0 ]
        then
                echo waiting on PIDS $pids 1>&2
                while [ $(echo $pids | xargs -n1 bash -c 'test -e /proc/${0:=NOTAPID} && echo' | wc -l) -gt 0 ]
                do
                        echo one of PIDS $pids up 1>&2
                        sleep $POLLING_INTERVAL
                done
                last_alive=$(date +%s)
        elif [ $(($last_alive + $SHUTDOWN_IDLE_SECONDS)) -gt $(date +%s) ]
        then
                echo no PIDS, sleeping $POLLING_INTERVAL 1>&2
                sleep $POLLING_INTERVAL
        else
                echo no PIDS for $SHUTDOWN_IDLE_SECONDS, shutting down 1>&2
                sudo shutdown -h 0
        fi
done

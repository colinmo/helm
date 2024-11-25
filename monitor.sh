#!/bin/bash
OLD=""
while true
    do
        PID="$(pidof hq)"
        if [[ $PID != $OLD ]]; then
            echo ""
            echo "   New process started."
            echo ""
        fi
        count="$(lsof -n | grep $PID | grep -i tcp | wc -l)"
        echo "$count"
        OLD="$PID"
        sleep 2
done
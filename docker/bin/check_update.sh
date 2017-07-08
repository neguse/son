#!/bin/sh

cd /volume/go/src/github.com/neguse/son

LOCAL=$(git rev-parse '@')
REMOTE=$(git rev-parse '@{u}')

if [ $LOCAL != $REMOTE ]; then
    /bin/build.sh
    s6-svc -t /var/run/s6/services/10-son
else

fi
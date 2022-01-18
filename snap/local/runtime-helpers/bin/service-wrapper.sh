#!/bin/bash -e

PROFILE_OPT=""

profile=$(snapctl get "profile")

if [ -n "$profile" ] && [ "$profile" != "default" ]; then
    PROFILE_OPT="-profile $profile"
    export SECRETSTORE_TOKENFILE="$SNAP_DATA/app-$profile/secrets-token.json"
fi

$SNAP/bin/app-service-configurable -confdir $SNAP_DATA/config/res $PROFILE_OPT -cp -r

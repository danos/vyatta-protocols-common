#!/bin/bash
local -a params
params="$( /opt/vyatta/sbin/vyatta-policy.pl --list-community extcommunity-list )"
echo -n "${params[@]##*/}"

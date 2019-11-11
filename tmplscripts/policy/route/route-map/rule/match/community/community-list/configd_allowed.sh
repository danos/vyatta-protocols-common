#!/bin/bash
local -a params
params="$( /opt/vyatta/sbin/vyatta-policy.pl --list-community community-list )"
echo -n "${params[@]##*/}"

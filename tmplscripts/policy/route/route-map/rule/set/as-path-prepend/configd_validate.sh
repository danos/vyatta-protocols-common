#!/opt/vyatta/bin/cliexec

# Warn the user that only 1 of the 3 options (as-path-prepend|own-as|last-as)
# can be configured at the same time

if [[ "$VAR(../prepend-as/own-as)" != ""  || "$VAR(../prepend-as/last-as)" != "" ]]; then
    echo "WARNING: You should not configure 'as-path-prepend' with 'prepend-as last-as' / 'prepend-as own-as'."
fi

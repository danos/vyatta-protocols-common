#!/usr/bin/perl

# **** License ****
# Copyright (c) 2019 AT&T Intellectual Property. All rights reserved.
# Copyright (c) 2014-2015 by Brocade Communications Systems, Inc.
# All rights reserved.
#
# SPDX-License-Identifier: GPL-2.0-only
# **** End License

use strict;
use warnings;

my @as_list = split( ' ', $ARGV[0] );
foreach my $as (@as_list) {
    exit 1 if ( $as =~ /[^\d\s]/ || $as < 1 || $as > 4294967295 );
}

die "Error: max 24 as path\n" if ( scalar(@as_list) > 24 );

exit 0;

#!/usr/bin/perl

# Copyright (c) 2019 AT&T Intellectual Property. All rights reserved.
# Copyright (c) 2014-2016 by Brocade Communications Systems, Inc.
# All rights reserved.
#
# SPDX-License-Identifier: GPL-2.0-only
#
# This code was originally developed by Vyatta, Inc.
# Portions created by Vyatta are Copyright (C) 2008 Vyatta, Inc.
# All Rights Reserved.

use strict;
use warnings;
use NetAddr::IP;
use Getopt::Long;

my ( $prefix, $clist_community, $rmap_community );

# Allowed well-know community values (see set community)
my %communities = (
    'internet'   => 1,
    'local-as'   => 1,
    'no-advertise' => 1,
    'no-export'  => 1,
);

GetOptions(
    "check-prefix-boundry=s" =>   \$prefix,
    "check-clist-community"  =>   \$clist_community,
    "check-rmap-community"   =>   \$rmap_community,
);

check_clist_community(@ARGV)   if ($clist_community);
check_rmap_community(@ARGV)    if ($rmap_community);
check_prefix_boundry($prefix)  if ($prefix);

exit 0;

sub check_prefix_boundry {
    my $prefix = shift;
    my ( $net, $network, $cidr );

    $net     = new NetAddr::IP $prefix;
    $network = $net->network()->cidr();
    $cidr    = $net->cidr();

    die "Your prefix must fall on a natural network boundry.  ",
      "Did you mean $network?\n"
      if ( $cidr ne $network );

    exit 0;
}

sub check_clist_community {
    foreach my $arg (@_) {
        if ($arg =~ /(\d+):(\d+)/) {
            # only allow non-zero ASID < 0xFFFF
            next if ($1 > 0 && $1 < 65535 && $2 < 65536);
        }
	next if $communities{$arg};

	die "$arg unknown community value\n"
    }
}

sub check_rmap_community {
    my $arg_count = $#ARGV + 1 ;
    foreach my $arg (@_) {
        if ($arg eq "none") {
            if ($arg_count > 1){
                die "cannot configure 'none' with other attributes\n";
	    } else {
                next;
            }
        }

        if ($arg =~ /(\d+):(\d+)/) {
            # only allow non-zero ASID < 0xFFFF
            next if ($1 > 0 && $1 < 65535 && $2 < 65536);
        }
	next if $communities{$arg};

	die "$arg unknown community value\n"
    }
}

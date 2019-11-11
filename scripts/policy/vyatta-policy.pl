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
use lib "/opt/vyatta/share/perl5/";
use Vyatta::Config;
use Vyatta::Misc;
use Getopt::Long;

my ( $clisttype );
my ( $routemap, $deleteroutemap, $listpolicy, $listcommunity );
my ( $checkcommlist , $checkextcommlist );
my ( $oplistcommunity );

GetOptions(
    "comm-list-type=s"               => \$clisttype,
    "check-routemap-action=s"        => \$routemap,
    "check-delete-routemap-action=s" => \$deleteroutemap,
    "list-policy=s"		     => \$listpolicy,
    "list-community=s"		     => \$listcommunity,
    "check-community-list=s"         => \$checkcommlist,
    "check-extcommunity-list=s"      => \$checkextcommlist,
    "op-list-community=s"              => \$oplistcommunity,
) or exit 1;

check_routemap_action($routemap)              if ($routemap);
check_delete_routemap_action($deleteroutemap) if ($deleteroutemap);
list_policy($listpolicy)	    	      if ($listpolicy);
list_community($listcommunity)  	      if ($listcommunity);
op_list_community($oplistcommunity)  	      if ($oplistcommunity);
check_community_list($checkcommlist, $clisttype, "community-list")        if ($checkcommlist);
check_community_list($checkextcommlist, $clisttype, "extcommunity-list")  if ($checkextcommlist);

exit 0;


## check_routemap_action
# check if the action has been changed since the last commit.
# we need to do this because routing will wipe the entire config if
# the action is changed.
# $1 = policy route-map <name> rule <num> action
sub check_routemap_action {
    my $routemap = shift;
    my $config   = new Vyatta::Config;

    my $action    = $config->setLevel("$routemap");
    my $origvalue = $config->returnOrigValue();
    if ($origvalue) {
        my $value = $config->returnValue();
        if ( "$value" ne "$origvalue" ) {
            exit 1;
        }
    }

    exit 0;
}

## check_delete_routemap_action
# don't allow deleteing the route-map action if other sibling nodes exist.
# action is required for all other route-map definitions
# $1 = policy route-map <name> rule <num>
sub check_delete_routemap_action {
    my $routemap = shift;
    my $config   = new Vyatta::Config;

    my @nodes = $config->listNodes("$routemap");

    exit(@nodes) ? 1 : 0;
}

## list available policies
sub list_policy {
   my $policy = shift;
   my $config = new Vyatta::Config;

   $config->setLevel("policy route $policy");
   my @nodes = $config->listNodes();
   foreach my $node (@nodes) { print "$node "; }
   return;
}

## list available communities
sub list_community_gen {
   my $community = shift;
   my $function = shift;
   my $config = new Vyatta::Config;

   $config->setLevel("policy route $community standard");
   my @nodes1 = $config->$function();
   $config->setLevel("policy route $community expanded");
   my @nodes2 = $config->$function();

   foreach my $node (@nodes1) { print "$node "; }
   foreach my $node (@nodes2) { print "$node "; }
   return;
}

## list available communities while configuring
sub list_community {
   my $policy = shift;
   list_community_gen($policy, "listNodes");
   return;
}

## list available communities while operational mode
sub op_list_community {
   my $policy = shift;
   list_community_gen($policy, "listOrigNodes");
   return;
}

#check if a community-list is already configured as different type
sub check_community_list {
    my ( $listval, $listtype, $list ) = @_;
    my $config = new Vyatta::Config;

    if ( $config->exists("policy route $list $listtype $listval") ) {
        print "Warning: Cannot configure $list $listval as both standard and expanded\n";
    }

    exit 0;
}

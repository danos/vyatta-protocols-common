#!/usr/bin/python3
#
# Copyright (c) 2018-2019, AT&T Intellectual Property.  All rights reserved.
#
# SPDX-License-Identifier: GPL-2.0-only

# This script is run as a service whenever there is a change to the static
# arp configuration. It parses the desired config and compares that with
# what is currently in the kernel, then updates the kernel to match.
# If there is no static arp configuration left, it exits, otherwise it
# monitors link state changes so that it can replace neighbors that get
# deleted from the kernel by a link flap.
#
# Kernel neighbor table is keyed on IP address + interface, routing instance
# is ignored. Vyatta yang is keyed on IP address and routing instance, so it
# possible to configure same neighor but with different h/w address in
# different routing instances. The last such instance encountered will be the
# one which takes effect.

import argparse
import json
import re
import os
import subprocess
import sys

debug = False


def debug_print(info):
    if debug:
        print(info, flush=True)


# Add protocols static arp configuration to dictionary
def arp_config_protocols(ri_name, proto, arp):
    if "static" not in proto:
        debug_print('no static in {} {}'.format(ri_name, proto))
        return
    static = proto["static"]
    if "arp" not in static:
        debug_print('no arp in {} {}'.format(ri_name, static))
        return
    arp_arr = static["arp"]
    for arp_entry in arp_arr:
        if "tagnode" not in arp_entry:
            debug_print('missing tagnode in {} {}'.format(ri_name, arp_entry))
            continue
        if not arp_entry["hwaddr"]:
            debug_print('missing hwaddr in {} {}'.format(ri_name, arp_entry))
            continue
        if not arp_entry["interface"]:
            debug_print('missing interface in {} {}'
                        .format(ri_name, arp_entry))
            continue
        key = arp_entry["tagnode"] + "," + arp_entry["interface"]
        arp[key] = arp_entry


# Add routing-instance static arp configuration to dictionary
def arp_config_routing(routing, arp):
    if "routing-instance" not in routing:
        debug_print('no routing-instance in {}'.format(routing))
        return
    ri_arr = routing["routing-instance"]
    for ri_entry in ri_arr:
        if "instance-name" not in ri_entry:
            debug_print('missing instance-name in {}'.format(ri_entry))
            continue
        ri_name = ri_entry["instance-name"]
        if "protocols" in ri_entry:
            arp_config_protocols(ri_name, ri_entry["protocols"], arp)


# Return a dictionary of the static arp table from the JSON configuration
def arp_config(arp_json):
    arp = {}
    if "protocols" in arp_json:
        arp_config_protocols("default", arp_json["protocols"], arp)
    if "routing" in arp_json:
        arp_config_routing(arp_json["routing"], arp)
    return arp


# Return a dictionary of contents from JSON file
def load_config(filename):
    debug_print('open: {}'.format(filename))
    try:
        with open(filename, "r") as arp_fd:
            arp_json = json.load(arp_fd)
            return arp_config(arp_json)
    except Exception as e:
        print(e, flush=True)
        return {}


# Return a dictionary of the current ip neighbor table
def ip_neighbors():
    cmd = ['ip', 'neigh', 'show', 'nud', 'permanent']
    try:
        p = subprocess.run(cmd, stdout=subprocess.PIPE,
                           universal_newlines=True)
    except Exception as e:
        print(e, flush=True)
        return {}

    out = p.stdout
    debug_print('ip neigh show: {}'.format(out))
    neigh = re.findall(
        r'^(\d+\.\d+\.\d+\.\d+)\s+dev\s+(\S+)\s+lladdr\s+(\S+)\s+PERMANENT',
        out, re.M)
    debug_print('extracted neighbors: {}'.format(neigh))
    arp = {}
    for (ip, dev, lladdr) in neigh:
        arp_entry = {'tagnode': ip, 'interface': dev, 'hwaddr': lladdr}
        key = ip + "," + dev
        arp[key] = arp_entry
    return arp


# Apply config changes from difference between current and desired arp tables
def process_config(new_arp, intf):
    cur_arp = ip_neighbors()

    debug_print('make arp changes from: {} to: {}'.format(cur_arp, new_arp))

    # Remove current entries that are not present in new set
    for key, cur_entry in cur_arp.items():
        if intf and intf != cur_entry["interface"]:
            debug_print('skip: {} != {}'.format(intf, cur_entry["interface"]))
            continue
        if key in new_arp:
            new_entry = new_arp[key]
        else:
            new_entry = {}
        debug_print('compare: {} vs {}'.format(cur_entry, new_entry))
        if cur_entry != new_entry:
            ip = cur_entry["tagnode"]
            dev = cur_entry["interface"]
            print('ip neigh delete {} dev {}'.format(ip, dev), flush=True)
            cmd = ['ip', 'neigh', 'delete', ip, 'dev', dev]
            try:
                subprocess.run(cmd)
            except Exception as e:
                print(e, flush=True)

    # Add new entries that are not present in current set
    for key, new_entry in new_arp.items():
        if intf and intf != new_entry["interface"]:
            debug_print('skip: {} != {}'.format(intf, new_entry["interface"]))
            continue
        if key in cur_arp:
            cur_entry = cur_arp[key]
        else:
            cur_entry = {}
        debug_print('compare: {} vs {}'.format(cur_entry, new_entry))
        if cur_entry != new_entry:
            ip = new_entry["tagnode"]
            lladdr = new_entry["hwaddr"]
            dev = new_entry["interface"]
            print('ip neigh replace {} lladdr {} dev {}'
                  .format(ip, lladdr, dev),
                  flush=True)
            cmd = ['ip', 'neigh', 'replace', ip, 'lladdr', lladdr, 'dev', dev]
            try:
                subprocess.run(cmd)
            except Exception as e:
                print(e, flush=True)


def main():
    parser = argparse.ArgumentParser(description='static arp config daemon')
    parser.add_argument('-d', '--debug', action='store_true',
                        help='turn on debugging')
    parser.add_argument('-l', '--logfile', type=argparse.FileType('a'),
                        help='send output to logfile')
    parser.add_argument('configfile', help='configuration file')
    args = parser.parse_args()
    global debug
    debug = args.debug
    if args.logfile:
        try:
            sys.stdout = open(args.logfile, "a")
        except Exception as e:
            print(e, flush=True)

    # Load in and process the config
    new_arp = load_config(args.configfile)
    process_config(new_arp, None)

    # If there is config, listen for link changes and re-process on link up.
    if new_arp != {}:
        print('listening for link up events', flush=True)
        cmd = ['ip', 'monitor', 'link']
        try:
            p = subprocess.Popen(cmd, stdout=subprocess.PIPE,
                                 universal_newlines=True)
        except Exception as e:
            print(e, flush=True)
            return 1

        while True:
            out = p.stdout.readline()
            if out == "":
                print('EOF from ip monitor')
                return 1
            debug_print('ip monitor: {}'.format(out))
            match = re.match(r'\d+: ([^\s:@]+).*state UP', out)
            if match:
                dev = match.group(1)
                debug_print('process config for up dev: {}'.format(dev))
                process_config(new_arp, dev)

    print('No static arp configuration - exiting.', flush=True)
    return 0

if __name__ == '__main__':
    exit(main())

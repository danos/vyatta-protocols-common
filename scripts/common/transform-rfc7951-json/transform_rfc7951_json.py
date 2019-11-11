#!/usr/bin/python3
#
# Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
#
# SPDX-License-Identifier: GPL-2.0-only

import json
import argparse
import sys

MODULE_PREFIX_SEP = ":"

print_err = lambda s : print(s, file=sys.stderr)

def transform_dict_keys(d):
    assert isinstance(d, dict)
    new_d = {}

    for key, val in d.items():
        if MODULE_PREFIX_SEP in key:
            key = key.split(MODULE_PREFIX_SEP, 1)[1]

        if val == [None]:
            val = None

        new_d[key] = val

    return new_d

def transform_json_file(f):
    try:
        json_d = json.load(f, object_hook=transform_dict_keys)
    except Exception as e:
        print_err("JSON decode error: {}".format(e))
        return None

    try:
        return json.dumps(json_d)
    except ValueError as e:
        print_err("JSON encode error: {}".format(e))

    return None

def main():
    parser = argparse.ArgumentParser(
                formatter_class=argparse.RawDescriptionHelpFormatter,
                description="""Transform RFC 7951 JSON

This utility transforms RFC 7951 encoded JSON into standard configuration JSON.

The following changes are made:
  - All data up to the first : character inclusive is removed from all object keys
  - Arrays with a single null value are changed to a simple null value

For example:
  '{"foo:bar": [null] }' ----> '{"bar": null}'

A 0 exit code indicates success and the result is printed to STDOUT.

If --file is not specified JSON will be read from STDIN.
""")
    parser.add_argument("-f", "--file", dest="file",
                        help="File to read input JSON from",
                        type=argparse.FileType("r"),
                        nargs="?", default=sys.stdin)
    args = parser.parse_args()

    stripped_json = transform_json_file(args.file)
    args.file.close()

    if stripped_json is None:
        return 1

    print(stripped_json)
    return 0

if __name__ == "__main__":
    try:
        exit(main())
    except KeyboardInterrupt:
        print_err("Quitting")
        exit(1)

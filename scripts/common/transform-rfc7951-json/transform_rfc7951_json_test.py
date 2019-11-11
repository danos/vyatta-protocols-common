#!/usr/bin/python3
#
# Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
#
# SPDX-License-Identifier: GPL-2.0-only

from io import StringIO
import json
import unittest

from transform_rfc7951_json import transform_json_file, \
                                   transform_dict_keys

class StripPrefixesTestCase(unittest.TestCase):

    def setUp(self):
        self.maxDiff = None

    def test_transform_dict_keys(self):
        self.assertEqual({ "foo" : { "2:bar" : "baz" } },
            transform_dict_keys({ "foo" : { "2:bar" : "baz" } }))

        self.assertEqual({ "foo" : None },
            transform_dict_keys({ "1:foo" : [ None ] }))

        self.assertEqual({ "foo" : { "2:bar" : "baz" } },
            transform_dict_keys({ "1:foo" : { "2:bar" : "baz" } }))

        self.assertEqual({ "2:3:foo" : { "bar" : "baz" } },
            transform_dict_keys({ "1:2:3:foo" : { "bar" : "baz" } }))

        self.assertEqual({ "foo" : { "2:bar" : "3:baz" } },
            transform_dict_keys({ "1:foo" : { "2:bar" : "3:baz" } }))

        self.assertEqual({
                            "foo" : { "bar" : "baz" },
                            "bar" : { "baz" : "bar" }
                         },
                         transform_dict_keys(
                            {
                            "1:foo" : { "bar" : "baz" },
                            "2:bar" : { "baz" : "bar" }
                            })
                        )

        self.assertEqual({
                            "foo" : { "bar" : "baz" },
                            "bar" : { "baz" : "bar" }
                         },
                         transform_dict_keys(
                            {
                            "3:foo" : { "bar" : "baz" },
                            "bar"   : { "baz" : "bar" }
                            })
                        )

    def test_transform_json_file(self):
        unstripped_json = StringIO("""
{
    "vyatta-interfaces-v1:interfaces": {
        "vyatta-interfaces-dataplane-v1:dataplane": [
            {
                "ip": {
                    "vyatta-protocols-pim-v1:pim": {
                        "hello-holdtime": 105,
                        "hello-interval": 30,
                        "mode": "sparse"
                    }
                },
                "tagnode": "dp0p1s1"
            },
            {
                "ipv6": {
                    "vyatta-protocols-pim6-v1:pim": {
                        "hello-holdtime": 105,
                        "hello-interval": 30,
                        "mode": "dense"
                    }
                },
                "tagnode": "dp0p1s2"
            }
        ]
    },
    "vyatta-protocols-v1:protocols": {
        "vyatta-protocols-pim-v1:pim": {
            "log": {
                "all": [
                    null
                ],
                "timer": {
                    "all": [
                        null
                    ],
                    "assert": {
                        "at": [
                            null
                        ]
                    }
                }
            },
            "register-suppression-timer": 60
        },
        "vyatta-protocols-pim6-v1:pim6": {
            "register-suppression-timer": 60
        }
    }
}""")

        expected_stripped_json = """
{
    "interfaces": {
        "dataplane": [
            {
                "ip": {
                    "pim": {
                        "hello-holdtime": 105,
                        "hello-interval": 30,
                        "mode": "sparse"
                    }
                },
                "tagnode": "dp0p1s1"
            },
            {
                "ipv6": {
                    "pim": {
                        "hello-holdtime": 105,
                        "hello-interval": 30,
                        "mode": "dense"
                    }
                },
                "tagnode": "dp0p1s2"
            }
        ]
    },
    "protocols": {
        "pim": {
            "log": {
                "all": null,
                "timer": {
                    "all": null,
                    "assert": {
                        "at": null
                    }
                }
            },
            "register-suppression-timer": 60
        },
        "pim6": {
            "register-suppression-timer": 60
        }
    }
}"""

        stripped_json = transform_json_file(unstripped_json)
        stripped_json_d = json.loads(stripped_json)
        expected_stripped_json_d = json.loads(expected_stripped_json)
        self.assertEqual(stripped_json_d, expected_stripped_json_d)

if __name__ == '__main__':
    unittest.main()

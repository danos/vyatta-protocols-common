// Copyright (c) 2018-2019, AT&T Intellectual Property.  All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package protocols_test

import (
	"bytes"
	"encoding/json"
	"eng.vyatta.net/protocols"
	"testing"
)

func TestConvertInterfaceConfig(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "dataplane" : [
         {
            "tagnode" : "dp0s4",
	    "ip" : {
	       "multicast" : {
		  "ttl-threshold" : 78
	       }
	    },
	    "ipv6" : {
	       "multicast" : {
		  "ttl-threshold" : 90
	       }
	    }
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	expected_json := []byte(`{
   "interfaces" : [
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 78
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 90
            }
         },
         "tagnode" : "dp0s4"
      }
   ],
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertInterfaceConfigOnlyVifs(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "dataplane" : [
         {
            "vif" : [
               {
                  "tagnode" : 100,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 78
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 90
                     }
                  }
               },
               {
                  "tagnode" : 200,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 77
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 82
                     }
                  }
               }
            ],
            "tagnode" : "dp0s4"
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	expected_json := []byte(`{
   "interfaces" : [
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 78
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 90
            }
         },
         "tagnode" : "dp0s4.100"
      },
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 77
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 82
            }
         },
         "tagnode" : "dp0s4.200"
      }
   ],
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertSwitchInterfaceConfig(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "switch" : [
         {
            "name" : "sw1",
	    "ip" : {
	       "multicast" : {
		  "ttl-threshold" : 78
	       }
	    },
	    "ipv6" : {
	       "multicast" : {
		  "ttl-threshold" : 90
	       }
	    }
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	expected_json := []byte(`{
   "interfaces" : [
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 78
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 90
            }
         },
         "tagnode" : "sw1"
      }
   ],
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertInterfaceswitchVif(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "switch" : [
         {
            "vif" : [
               {
                  "tagnode" : 100,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 79
                     }
                  }
               }
            ],
            "name" : "sw1"
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)
	expected_json := []byte(`{
   "interfaces" : [
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 79
            }
         },
         "tagnode" : "sw1.100"
      }
   ],
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)
	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertInterfaceswitchVifInvalidName(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "switch" : [
         {
            "vif" : [
               {
                  "tagnode" : 100,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 79
                     }
                  }
               }
            ],
            "if-name" : "sw1"
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)
	expected_json := []byte(`{
   "interfaces" : null,
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)
	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertInterfaceConfigParentAndVifs(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "dataplane" : [
         {
            "vif" : [
               {
                  "tagnode" : 100,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 78
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 90
                     }
                  }
               },
               {
                  "tagnode" : 200,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 77
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 82
                     }
                  }
               }
            ],
	    "ip" : {
	       "multicast" : {
		  "ttl-threshold" : 100
	       }
	    },
	    "ipv6" : {
	       "multicast" : {
		  "ttl-threshold" : 110
	       }
	    },
            "tagnode" : "dp0s4"
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	expected_json := []byte(`{
   "interfaces" : [
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 78
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 90
            }
         },
         "tagnode" : "dp0s4.100"
      },
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 77
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 82
            }
         },
         "tagnode" : "dp0s4.200"
      },
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 100
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 110
            }
         },
         "tagnode" : "dp0s4"
      }
   ],
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertInterfaceConfigParentAndInvalidVif(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "dataplane" : [
         {
            "vif" : [
               {
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 78
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 90
                     }
                  }
               },
               {
                  "tagnode" : 200,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 77
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 82
                     }
                  }
               }
            ],
	    "ip" : {
	       "multicast" : {
		  "ttl-threshold" : 100
	       }
	    },
	    "ipv6" : {
	       "multicast" : {
		  "ttl-threshold" : 110
	       }
	    },
	    "tagnode" : "dp0s4"
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	expected_json := []byte(`{
   "interfaces" : [
      {
         "ip" : {
            "multicast" : {
               "ttl-threshold" : 77
            }
         },
         "ipv6" : {
            "multicast" : {
               "ttl-threshold" : 82
            }
         },
         "tagnode" : "dp0s4.200"
      },
      {
	 "ip" : {
	    "multicast" : {
	       "ttl-threshold" : 100
	    }
	 },
	 "ipv6" : {
	    "multicast" : {
	       "ttl-threshold" : 110
	    }
	 },
	 "tagnode" : "dp0s4"
      }
   ],
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func TestConvertInterfaceConfigInvalidParentAndVifs(t *testing.T) {
	input_json := []byte(`{
   "interfaces" : {
      "dataplane" : [
         {
            "vif" : [
               {
		  "tagnode" : 100,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 78
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 90
                     }
                  }
               },
               {
                  "tagnode" : 200,
                  "ip" : {
                     "multicast" : {
                        "ttl-threshold" : 77
                     }
                  },
                  "ipv6" : {
                     "multicast" : {
                        "ttl-threshold" : 82
                     }
                  }
               }
            ],
	    "ip" : {
	       "multicast" : {
		  "ttl-threshold" : 100
	       }
	    },
	    "ipv6" : {
	       "multicast" : {
		  "ttl-threshold" : 110
	       }
	    }
         }
      ]
   },
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	expected_json := []byte(`{
   "interfaces" : null,
   "protocols" : {
      "multicast" : {
         "ip" : {
            "routing" : null
         }
      }
   }
}`)

	runConvertInterfaceConfigTest(t, input_json, expected_json)
}

func runConvertInterfaceConfigTest(t *testing.T, input_json []byte, expected_json []byte) {
	var expected interface{}
	var output interface{}

	// unmarshal/marshal expected output to match output formatting
	err := json.Unmarshal(expected_json, &expected)
	if err != nil {
		t.Fatalf("%v", err)
	}

	expected_string, err := json.MarshalIndent(expected, "", "    ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = json.Unmarshal(input_json, &output)
	if err != nil {
		t.Fatalf("%v", err)
	}

	config := protocols.ConvertInterfaceConfig(output.(map[string]interface{}))

	actual_string, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	if bytes.Compare(actual_string, expected_string) != 0 {
		t.Fatalf(`configs do not match
expected:
%v

got:
%v
`, string(expected_string), string(actual_string))
	}
}

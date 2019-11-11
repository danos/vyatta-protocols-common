// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package static

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os/exec"
	"strconv"
)

func MapByKey(arr []interface{}, key_name string) map[string]map[string]interface{} {
	pmap := make(map[string]map[string]interface{})

	for _, entry := range arr {
		entry_map := entry.(map[string]interface{})
		key := fmt.Sprint(entry_map[key_name])
		pmap[key] = entry_map
	}

	return pmap
}

func IsNexthopDisabled(nh_map map[string]interface{}) bool {
	if nh_map == nil {
		return false
	}

	_, disabled := nh_map["disable"]
	return disabled
}

func TranslateNexthops(pmap map[string]interface{}, key string) {
	if pmap == nil || pmap[key] == nil {
		return
	}

	nh_arr := pmap[key].([]interface{})
	if nh_arr == nil {
		return
	}

	deleted := false

	//Reverse order so can delete disabled nexthops
	for i := len(nh_arr) - 1; i >= 0; i-- {
		nh_entry := nh_arr[i]
		nh_map := nh_entry.(map[string]interface{})

		//Remove from array if disabled
		if IsNexthopDisabled(nh_map) {
			nh_arr = append(nh_arr[:i], nh_arr[i+1:]...)
			deleted = true
		} else {
			//Key is "tagnode" in default routing-instance,
			//but "interface-name" in non-default.
			//Translate to the former for consistency.
			if nh_map["interface-name"] != nil {
				nh_map["tagnode"] = nh_map["interface-name"]
				delete(nh_map, "interface-name")
			}
		}
	}

	//If we deleted a nexthop, update the parent map
	if deleted {
		if len(nh_arr) == 0 {
			delete(pmap, key)
		} else {
			pmap[key] = nh_arr
		}
	}
}

func TranslateNexthopInstances(pmap map[string]interface{}, pkey string,
	nkey string) {
	if pmap == nil || pmap[pkey] == nil {
		return
	}

	inst_arr := pmap[pkey].([]interface{})
	if inst_arr == nil {
		return
	}

	deleted := false

	//Reverse order so can delete disabled instances
	for i := len(inst_arr) - 1; i >= 0; i-- {
		inst_entry := inst_arr[i]
		inst_entry_map := inst_entry.(map[string]interface{})
		TranslateNexthops(inst_entry_map, nkey)
		if inst_entry_map[nkey] == nil {
			inst_arr = append(inst_arr[:i], inst_arr[i+1:]...)
			deleted = true
		}
	}

	//If we deleted a nexthop, update the parent map
	if deleted {
		if len(inst_arr) == 0 {
			delete(pmap, pkey)
		} else {
			pmap[pkey] = inst_arr
		}
	}
}

func TranslateRoutes(pmap map[string]interface{}, pkey string, nkey string) {
	if pmap == nil || pmap[pkey] == nil {
		return
	}

	route_arr := pmap[pkey].([]interface{})
	if route_arr == nil {
		return
	}

	deleted := false

	//Reverse order so can delete disabled routes
	for i := len(route_arr) - 1; i >= 0; i-- {
		route_entry := route_arr[i]
		route_entry_map := route_entry.(map[string]interface{})

		//Translate next-hop[-interface]s
		TranslateNexthops(route_entry_map, nkey)
		nh_if := route_entry_map[nkey]
		if nh_if == nil {
			delete(route_entry_map, nkey)
		}

		//"next-hop-routing-instance" is used in default instance,
		//but "next-hop-routing-instance-v6" in non-default.
		//Translate to the former for consistency.
		if route_entry_map["next-hop-routing-instance-v6"] != nil {
			route_entry_map["next-hop-routing-instance"] =
				route_entry_map["next-hop-routing-instance-v6"]
			delete(route_entry_map, "next-hop-routing-instance-v6")
		}

		//Translate next-hop-routing-instances
		TranslateNexthopInstances(route_entry_map,
			"next-hop-routing-instance", nkey)
		inst_if := route_entry_map["next-hop-routing-instance"]
		if inst_if == nil {
			delete(route_entry_map, "next-hop-routing-instance")
		}

		//Delete route itself if only tagnode is defined
		if len(route_entry_map) <= 1 {
			route_arr = append(route_arr[:i], route_arr[i+1:]...)
			deleted = true
		}
	}

	//Delete parent if all routes have gone
	if deleted {
		if len(route_arr) == 0 {
			delete(pmap, pkey)
		} else {
			pmap[pkey] = route_arr
		}
	}
}

func TranslateTables(pmap, old_pmap map[string]interface{}, key string, ri string) {
	if pmap[key] == nil && old_pmap[key] == nil {
		return
	}

	var tbl_arr, old_tbl_arr []interface{}

	if pmap[key] == nil {
		tbl_arr = make([]interface{}, 0)
	} else {
		tbl_arr = pmap[key].([]interface{})
	}
	if old_pmap[key] == nil {
		old_tbl_arr = make([]interface{}, 0)
	} else {
		old_tbl_arr = old_pmap[key].([]interface{})
	}

	tbl_map := MapByKey(tbl_arr, "tagnode")
	old_tbl_map := MapByKey(old_tbl_arr, "tagnode")

	for table_id, tbl_entry_map := range tbl_map {
		exec.Command("/opt/vyatta/sbin/vrf-manager", "--add-table", ri, table_id).Run()

		out, err := exec.Command("/opt/vyatta/sbin/getvrftable", "--pbr-table", ri, table_id).Output()
		if err != nil {
			msg := "Failed to getvrftable: " + err.Error()
			log.Errorln(msg)
			return
		}
		log.Infoln("Translated table " + table_id + " to " + string(out))
		// Again, float - strange but true
		new_table_id, err := strconv.ParseFloat(string(out), 64)
		if err != nil {
			msg := "Bad format for table id: " + err.Error()
			log.Errorln(msg)
			return
		}
		tbl_entry_map["tagnode"] = new_table_id

		TranslateRoutes(tbl_entry_map, "interface-route",
			"next-hop-interface")
		TranslateRoutes(tbl_entry_map, "interface-route6",
			"next-hop-interface")
		TranslateRoutes(tbl_entry_map, "route", "next-hop")
		TranslateRoutes(tbl_entry_map, "route6", "next-hop")
	}

	for table_id, _ := range old_tbl_map {
		if tbl_map[table_id] == nil {
			log.Infoln("Requesting delete of table " + table_id + " in VRF " + ri)
			exec.Command("/opt/vyatta/sbin/vrf-manager", "--del-table", ri, table_id).Run()
		}
	}

}

func TranslateProtocols(proto_if, old_proto_if interface{}, ri string) {
	if proto_if == nil && old_proto_if == nil {
		return
	}

	var proto_map, old_proto_map map[string]interface{}
	if proto_if == nil {
		proto_map = make(map[string]interface{})
	} else {
		proto_map = proto_if.(map[string]interface{})
	}
	if old_proto_if == nil {
		old_proto_map = make(map[string]interface{})
	} else {
		old_proto_map = old_proto_if.(map[string]interface{})
	}

	if proto_map["static"] == nil && old_proto_map["static"] == nil {
		return
	}

	var static_map, old_static_map map[string]interface{}

	if proto_map["static"] == nil {
		static_map = make(map[string]interface{})
	} else {
		static_map = proto_map["static"].(map[string]interface{})
	}
	if old_proto_map["static"] == nil {
		old_static_map = make(map[string]interface{})
	} else {
		old_static_map = old_proto_map["static"].(map[string]interface{})
	}

	TranslateRoutes(static_map, "interface-route", "next-hop-interface")
	TranslateRoutes(static_map, "interface-route6", "next-hop-interface")
	TranslateRoutes(static_map, "route", "next-hop")
	TranslateRoutes(static_map, "route6", "next-hop")
	TranslateTables(static_map, old_static_map, "table", ri)
}

func TranslateRouting(routing_if, old_routing_if interface{}) {
	if routing_if == nil && old_routing_if == nil {
		return
	}

	var old_routing_map, routing_map map[string]interface{}

	if old_routing_if == nil {
		// make it valid for convenience
		old_routing_map = make(map[string]interface{})
	} else {
		old_routing_map = old_routing_if.(map[string]interface{})
	}

	if routing_if == nil {
		// make it valid for convenience
		routing_map = make(map[string]interface{})
	} else {
		routing_map = routing_if.(map[string]interface{})
	}

	var ri_arr, old_ri_arr []interface{}

	if routing_map["routing-instance"] == nil &&
		old_routing_map["routing-instance"] == nil {
		return
	}

	if routing_map["routing-instance"] == nil {
		ri_arr = make([]interface{}, 0)
	} else {
		ri_arr = routing_map["routing-instance"].([]interface{})
	}
	if old_routing_map["routing-instance"] == nil {
		old_ri_arr = make([]interface{}, 0)
	} else {
		old_ri_arr = old_routing_map["routing-instance"].([]interface{})
	}

	ri_map_by_key := MapByKey(ri_arr, "instance-name")
	old_ri_map_by_key := MapByKey(old_ri_arr, "instance-name")

	for key, ri_entry_map := range ri_map_by_key {
		old_ri_entry_map := old_ri_map_by_key[key]
		if old_ri_entry_map == nil {
			old_ri_entry_map = make(map[string]interface{})
		}
		TranslateProtocols(ri_entry_map["protocols"],
			old_ri_entry_map["protocols"], key)
	}
	for key, old_ri_entry_map := range old_ri_map_by_key {
		ri_entry := ri_map_by_key[key]
		if ri_entry == nil {
			TranslateProtocols(nil,
				old_ri_entry_map["protocols"], key)
		}
	}
}

func Translate(frontend_map, old_frontend_map map[string]interface{}) {
	if frontend_map != nil || old_frontend_map != nil {
		TranslateProtocols(frontend_map["protocols"],
			old_frontend_map["protocols"], "default")
		TranslateRouting(frontend_map["routing"],
			old_frontend_map["routing"])
	}
}

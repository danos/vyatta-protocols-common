// Copyright (c) 2017-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package protocols

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/danos/encoding/rfc7951"
	"github.com/danos/vci"
	multierr "github.com/hashicorp/go-multierror"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
)

const (
	cfgDir      = "/etc/vyatta-routing"
	cfgNotifDir = "/run/routing/config"
)

/* Common JSON configuration object keys */
const (
	INTERFACES_KEY             = "interfaces"
	INTERFACES_LIST_KEY        = "tagnode"
	SWITCH_INTERFACES_LIST_KEY = "name"
	VIF_KEY                    = "vif"
)

var interface_list_keys = [...]string{INTERFACES_LIST_KEY, SWITCH_INTERFACES_LIST_KEY}

func init() {
	formatter := &log.TextFormatter{}
	formatter.DisableTimestamp = true
	formatter.DisableSorting = true

	log.SetFormatter(formatter)
	log.SetLevel(log.WarnLevel)
}

type ProtocolsModelComponentConfig struct {
	Interfaces []struct {
		Tagnode string                 `json:"tagnode,omitempty"`
		Name    string                 `json:"name,omitempty"`
		Ip      map[string]interface{} `json:"ip,omitempty"`
		Ipv6    map[string]interface{} `json:"ipv6,omitempty"`
	} `json:"interfaces,omitempty"`
	Protocols map[string]interface{} `json:"protocols,omitempty"`
	Routing   struct {
		RoutingInstance []struct {
			InstanceName string                 `json:"instance-name,omitempty"`
			Protocols    map[string]interface{} `json:"protocols,omitempty"`
		} `json:"routing-instance,omitempty"`
	} `json:"routing,omitempty"`
}

func ParseJsonComponentConfig(cfg []byte) (*ProtocolsModelComponentConfig, error) {
	var pmc_cfg ProtocolsModelComponentConfig

	err := json.Unmarshal(cfg, &pmc_cfg)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	return &pmc_cfg, nil
}

type CommonArgs struct {
	User string
}

type ProtocolsModelComponentCheckFunc func(*ProtocolsModelComponent, []byte) error
type ProtocolsModelComponentGetFunc func(*ProtocolsModelComponent) []byte
type ProtocolsModelComponentSetFunc func(*ProtocolsModelComponent, []byte) error
type ProtocolsModelComponentMeaningfulConfigFunc func(*ProtocolsModelComponent, []byte) bool

/*
 * The ProtocolsModelComponent structure is the building block of a
 * VCI component for a particular routing daemon.
 *
 * It implements utility functions for receiving and translating configuration,
 * writing configuration files, and more.
 *
 * A VCI component for a routing daemon should call NewProtocolsModelComponent()
 * to obtain a new instance, setting this as its VCI component model config.
 */
type ProtocolsModelComponent struct {
	modelName      string
	model          vci.Model
	component      vci.Component
	configFileName string
	daemons        map[string]*ProtocolsDaemon
	checkFunc      ProtocolsModelComponentCheckFunc
	getFunc        ProtocolsModelComponentGetFunc
	setFunc        ProtocolsModelComponentSetFunc
	meanFunc       ProtocolsModelComponentMeaningfulConfigFunc
	args           *CommonArgs
}

func NewProtocolsModelComponent(
	componentName, modelName, configFileName string,
) *ProtocolsModelComponent {

	pmc := &ProtocolsModelComponent{}
	pmc.daemons = make(map[string]*ProtocolsDaemon)
	pmc.modelName = modelName
	pmc.configFileName = configFileName
	pmc.setFunc = defaultPmcSetFunc
	pmc.meanFunc = defaultPmcMeanFunc
	pmc.args = ParseCommonArgs()

	if pmc.GetDaemonConfigFilePath() == pmc.GetSystemConfigFilePath() {
		panic("Daemon and system config file paths are identical!")
	}

	pmc.component = vci.NewComponent(componentName)
	pmc.model = pmc.component.Model(pmc.GetModelName())
	pmc.model.Config(pmc)

	return pmc
}

func (pmc *ProtocolsModelComponent) Run() error {
	err := pmc.component.Run()
	if err != nil {
		return err
	}

	log.Infoln("Ready")

	ret := pmc.component.Wait()

	log.Infoln("Shutting down")

	/*
	 * If any of the component's daemons are scheduled to be shutdown
	 * then stop them now.
	 */
	for _, pd := range pmc.daemons {
		pd.LockControl()
		pd.StopAndDisableIfScheduled()
		pd.UnlockControl()
	}

	return ret
}

func (pmc *ProtocolsModelComponent) SetRPC(rpcName string, rpc interface{}) {
	pmc.model.RPC(rpcName, rpc)
}

func (pmc *ProtocolsModelComponent) SetCheckFunction(checkFunc ProtocolsModelComponentCheckFunc) {
	pmc.checkFunc = checkFunc
}

func (pmc *ProtocolsModelComponent) SetGetFunction(getFunc ProtocolsModelComponentGetFunc) {
	pmc.getFunc = getFunc
}

func (pmc *ProtocolsModelComponent) SetSetFunction(setFunc ProtocolsModelComponentSetFunc) {
	pmc.setFunc = setFunc
}

func (pmc *ProtocolsModelComponent) SetMeaningfulConfigFunction(meanFunc ProtocolsModelComponentMeaningfulConfigFunc) {
	pmc.meanFunc = meanFunc
}

func (pmc *ProtocolsModelComponent) AddDaemon(pd *ProtocolsDaemon) {
	pmc.daemons[pd.GetUnitName()] = pd
}

func (pmc *ProtocolsModelComponent) GetDaemon(daemon string) *ProtocolsDaemon {
	pd, _ := pmc.daemons[daemon]
	return pd
}

/*
 * Returns the path to the watched daemon notification file
 */
func (pmc *ProtocolsModelComponent) GetDaemonNotificationFilePath() string {
	return path.Join(cfgNotifDir, pmc.configFileName)
}

/*
 * Returns the path to the daemon configuration file
 */
func (pmc *ProtocolsModelComponent) GetDaemonConfigFilePath() string {
	return path.Join(cfgDir, pmc.configFileName)
}

/*
 * Returns the path to the system configuration file which contains the
 * pristine configuration received from the configuration system.
 */
func (pmc *ProtocolsModelComponent) GetSystemConfigFilePath() string {
	return path.Join(cfgDir, pmc.modelName+".json")
}

/*
 * Returns the model name of this component
 */
func (pmc *ProtocolsModelComponent) GetModelName() string {
	return pmc.modelName
}

/*
 * VCI Get implementation
 *
 * If a get callback function has not been set, this function returns
 * the current contents of the system configuration file. Otherwise the
 * get callback is invoked.
 */
func (pmc *ProtocolsModelComponent) Get() []byte {
	if pmc.getFunc != nil {
		return pmc.getFunc(pmc)
	}

	cfg, _ := pmc.GetSystemConfig()
	return cfg
}

/*
 * VCI Check implementation
 *
 * This function checks (validates) the JSON configuration, after first
 * converting it to our internal format.
 *
 * If a check callback function has been set it is called and its result
 * returned. Otherwise this function is a no-op.
 */
func (pmc *ProtocolsModelComponent) Check(cfg []byte) error {
	if pmc.checkFunc == nil {
		return nil
	}

	cfg, err := ConvertConfigToInternalJson(cfg)
	if err != nil {
		return err
	}

	return pmc.checkFunc(pmc, cfg)
}

/*
 * VCI Set implementation
 *
 * This function receives and checks the RFC 7951 JSON configuration,
 * updates the cached system configuration file then decodes the config
 * into standard JSON.
 *
 * If a set function callback has been defined this is then invoked. This
 * callback is responsible for manipulating the configuration as required
 * and notifying its routing daemon of the new configuration.
 *
 * If a set callback is not defined the stripped configuration is simply
 * written to the daemon configuration file.
 */
func (pmc *ProtocolsModelComponent) Set(cfg []byte) error {
	log.Infoln("Setting Config")

	conv_cfg, err := ConvertConfigToInternalJson(cfg)
	if err != nil {
		return err
	}

	ret_err := NewMultiError()

	/*
	 * Hand off to the Set handler
	 */
	ret_err = multierr.Append(ret_err, pmc.setFunc(pmc, conv_cfg))

	/*
	 * Cache the received system configuration
	 */
	pmc.SetSystemConfig(cfg)

	/*
	 * Enable and start daemons if we have config, otherwise stop and disable them
	 */
	for _, pd := range pmc.daemons {
		if pmc.meanFunc(pmc, conv_cfg) {
			pd.LockControl()

			ret_err = multierr.Append(ret_err, pd.Enable())
			ret_err = multierr.Append(ret_err, pd.Start())

			pd.UnlockControl()
		} else {
			pd.LockControl()
			pd.ScheduleStopAndDisable()
			pd.UnlockControl()
		}
	}

	return ret_err.ErrorOrNil()
}

/*
 * Convert JSON configuration received from the configuration system into
 * the internal format expected by the ProtocolsModelComponent set, get, and
 * check callbacks.
 *
 * The input JSON is converted from RFC 7951 format, and interface configuration
 * is restructured, then returned as a new byte array.
 */
func ConvertConfigToInternalJson(cfg []byte) ([]byte, error) {
	conv_cfg, err := ConvertFromRfc7951Json(cfg)
	if err != nil {
		log.Errorln("Failed to convert from RFC 7951 JSON: " + err.Error())
		return conv_cfg, err
	}

	conv_cfg, err = ConvertJsonInterfaceConfig(conv_cfg)
	if err != nil {
		log.Errorln("Failed to translate interface configuration: " + err.Error())
	}

	return conv_cfg, err
}

/*
 * Translates any interface configuration into the expected format
 *
 * cfg should contain non-RFC 7951 encoded JSON
 */
func ConvertJsonInterfaceConfig(cfg []byte) ([]byte, error) {
	var cfg_interface interface{}

	err := json.Unmarshal(cfg, &cfg_interface)
	if err != nil {
		log.Errorln("Failed to unmarshal configuration: " + err.Error())
		return EmptyConfig(), err
	}

	cfg_map := ConvertInterfaceConfig(cfg_interface.(map[string]interface{}))

	cfg, err = json.Marshal(cfg_map)
	if err != nil {
		log.Errorln("Failed to marshal configuration: " + err.Error())
		return EmptyConfig(), err
	}

	return cfg, err
}

/*
 * Default ProtocolsModelComponent Set handler
 *
 * Pretty formats the JSON configuration and writes it to the daemon
 * configuration file.
 */
func defaultPmcSetFunc(pmc *ProtocolsModelComponent, cfg []byte) error {
	return pmc.WriteDaemonConfig(FormatJson(cfg))
}

func defaultPmcMeanFunc(pmc *ProtocolsModelComponent, cfg []byte) bool {
	return !IsEmptyConfig(cfg)
}

/*
 * Notify the daemon to reload its configuration
 */
func (pmc *ProtocolsModelComponent) NotifyDaemon() error {
	/*
	 * Create/truncate the notification file to trigger the daemon to
	 * reload the configuration.
	 */
	file, err := os.Create(pmc.GetDaemonNotificationFilePath())
	if err != nil {
		return err
	}

	file.Close()
	return nil
}

/*
 * Writes a JSON file in the context of a ProtocolsModelComponent
 */
func (pmc *ProtocolsModelComponent) WriteJsonFile(json []byte, path string) error {
	return WriteJsonFile(string(json), path, pmc.args.User)
}

/*
 * Write the JSON contained in the cfg byte array to the daemon configuration file
 */
func (pmc *ProtocolsModelComponent) WriteDaemonConfig(cfg []byte) error {
	err := pmc.WriteJsonFile(cfg, pmc.GetDaemonConfigFilePath())
	if err != nil {
		return err
	}

	return pmc.NotifyDaemon()
}

/*
 * Write the RFC 7951 JSON contained in the cfg byte array to the cached
 * system configuration file
 */
func (pmc *ProtocolsModelComponent) SetSystemConfig(cfg []byte) error {
	return pmc.WriteJsonFile(cfg, pmc.GetSystemConfigFilePath())
}

/*
 * Returns the currently cached system configuration as RFC 7951 encoded JSON
 */
func (pmc *ProtocolsModelComponent) GetSystemConfig() ([]byte, error) {
	cfg, err := ioutil.ReadFile(pmc.GetSystemConfigFilePath())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorln("Failed to read system config file: " + err.Error())
		}
		return EmptyConfig(), err
	}

	return cfg, nil
}

/*
 * Pretty formats the input_json byte array into a new byte array
 */
func FormatJson(input_json []byte) []byte {
	var buffer bytes.Buffer
	json.Indent(&buffer, input_json, "", "    ")
	return buffer.Bytes()
}

/*
 * Decodes the RFC 7951 encoded JSON from the rfc7951_json byte array
 * and returns this as a new byte array.
 */
func ConvertFromRfc7951Json(rfc7591_json []byte) ([]byte, error) {
	var obj interface{}

	err := rfc7951.Unmarshal(rfc7591_json, &obj)
	if err != nil {
		return EmptyConfig(), err
	}

	decoded_json, err := json.Marshal(obj)
	if err != nil {
		return EmptyConfig(), err
	}

	tmpfile, err := ioutil.TempFile("", "rfc7951-json")
	if err != nil {
		return EmptyConfig(), err
	}

	defer os.Remove(tmpfile.Name()) // clean up on return

	if _, err := tmpfile.Write(decoded_json); err != nil {
		tmpfile.Close()
		return EmptyConfig(), err
	}

	if err := tmpfile.Close(); err != nil {
		return EmptyConfig(), err
	}

	strip_cmd := exec.Command("/opt/vyatta/bin/transform-rfc7951-json",
		"-f", tmpfile.Name())
	return strip_cmd.CombinedOutput()
}

/*
 * Modifies the configuration structure defined by cfg_map to remove the
 * interface type and move VIF configuration from beneath the respective
 * parent interface to the top level using the <parent>.<vif-num> name.
 *
 * If there is no INTERFACES_KEY entry in cfg_map then this function is a
 * no-op and cfg_map is simply returned.
 */
func ConvertInterfaceConfig(cfg_map map[string]interface{}) map[string]interface{} {
	var interface_cfg_list []interface{}

	if cfg_map[INTERFACES_KEY] == nil {
		return cfg_map
	}

	/*
	 * Loop over all interface types (dataplane, loopback etc) and insert
	 * the configuration for each interface into the interface_cfg_list
	 * array.
	 *
	 * Configuration for VIFs is also inserted into interface_cfg_list
	 * and the tagnode renamed to the <parent>.<num> format.
	 */
	for _, interface_types := range cfg_map[INTERFACES_KEY].(map[string]interface{}) {
		for _, interface_cfg := range interface_types.([]interface{}) {
			interface_cfg_map := interface_cfg.(map[string]interface{})

			var if_name_exists bool
			var if_name string
			// Get parent interface name
			for _, key := range interface_list_keys {
				if_name, if_name_exists = interface_cfg_map[key].(string)
				if if_name_exists {
					// convert interface to use 'tagnode'
					// key if required
					if key != INTERFACES_LIST_KEY {
						interface_cfg_map[INTERFACES_LIST_KEY] = if_name
						delete(interface_cfg_map, key)
					}

					break
				}
			}

			if !if_name_exists {
				continue
			}

			vif_list, _ := interface_cfg_map[VIF_KEY].([]interface{})

			// Loop over any VIFs
			for _, vif_cfg := range vif_list {
				vif_cfg_map := vif_cfg.(map[string]interface{})

				vif_num, vif_num_exists := vif_cfg_map[INTERFACES_LIST_KEY]
				if !vif_num_exists {
					continue
				}

				// Rename interface to <if_name>.<vif_num> format
				vif_cfg_map[INTERFACES_LIST_KEY] = GenerateVifName(if_name, vif_num)
				interface_cfg_list = append(interface_cfg_list, vif_cfg_map)
			}

			// Delete old VIF config from the parent
			delete(interface_cfg_map, VIF_KEY)

			// Add parent interface config if there is any
			if len(interface_cfg_map) > 1 {
				interface_cfg_list = append(interface_cfg_list, interface_cfg_map)
			}
		}
	}

	// Insert new interface config list under "interfaces" parent
	cfg_map[INTERFACES_KEY] = interface_cfg_list
	return cfg_map
}

/*
 * Returns a byte array representing an empty configuration
 */
func EmptyConfig() []byte {
	return []byte("{}")
}

func IsEmptyConfig(cfg []byte) bool {
	return bytes.Equal(bytes.Trim(cfg, "\n"), EmptyConfig())
}

/*
 * Returns a VIF interface name based on a parent interface name and VIF number
 */
func GenerateVifName(parent_name string, vif_num interface{}) string {
	return fmt.Sprintf("%v.%v", parent_name, vif_num)
}

func ParseCommonArgs() *CommonArgs {
	user := flag.String("user", "root", "User context of backend daemon")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	commonArgs := &CommonArgs{}
	commonArgs.User = *user
	return commonArgs
}

func writeJsonFile(json string, fileName string, owner string, perms os.FileMode) error {
	owner_user, err := user.Lookup(owner)
	if err != nil {
		log.Errorln(err)
		return err
	}

	uid, _ := strconv.Atoi(owner_user.Uid)
	gid, _ := strconv.Atoi(owner_user.Gid)

	tmpFileName := fileName + ".tmp"
	ioutil.WriteFile(tmpFileName, []byte(json), perms)

	ret_err := NewMultiError()

	err = os.Chown(tmpFileName, uid, gid)
	if err != nil {
		log.Errorln(err)
		ret_err = multierr.Append(ret_err, err)
	}

	err = os.Rename(tmpFileName, fileName)
	if err != nil {
		log.Errorln(err)
		ret_err = multierr.Append(ret_err, err)
	}

	return ret_err.ErrorOrNil()
}

func WriteJsonFile(json string, fileName string, owner string) error {
	return writeJsonFile(json, fileName, owner, 0600)
}

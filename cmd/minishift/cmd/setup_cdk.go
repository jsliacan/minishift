/*
Copyright (C) 2016 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"io/ioutil"
	"runtime"

	"github.com/docker/machine/libmachine"
	"github.com/golang/glog"
	"github.com/minishift/minishift/cmd/minishift/cmd/addon"
	configCmd "github.com/minishift/minishift/cmd/minishift/cmd/config"
	"github.com/minishift/minishift/cmd/minishift/state"
	addonbindata "github.com/minishift/minishift/out/bindata"
	"github.com/minishift/minishift/pkg/minikube/cluster"
	"github.com/minishift/minishift/pkg/minikube/constants"
	"github.com/minishift/minishift/pkg/minishift/addon/manager"
	minishiftConfig "github.com/minishift/minishift/pkg/minishift/config"
	minishiftConstants "github.com/minishift/minishift/pkg/minishift/constants"
	"github.com/minishift/minishift/pkg/util/os/atexit"
	"github.com/minishift/minishift/pkg/version"
	"github.com/minishift/minishift/setup-cdk/bindata"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	homeDirFlag       = "minishift-home"
	defaultDriverFlag = "default-vm-driver"
	forceInstallFlag  = "force"

	isoName        = "minishift-rhel7.iso"
	configFileName = "config.json"
	markerTmpl     = `openshift.auth.scheme=basic
openshift.auth.username=developer
openshift.auth.password=developer
cdk.version={{ . }}
`
	CDKMarker        = "cdk"
	CDKDefaultMemory = "4096"
)

var (
	homeDir              string
	defaultDriver        string
	forceInstall         bool
	defaultEnabledAddons = []string{"anyuid", "admin-user", "xpaas"}
	defaultAddons        = append(defaultEnabledAddons, "registry-route", "che", "htpasswd-identity-provider", "eap-cd")
)

// cdkSetupCmd represents the command to setup CDK 3 on the host
var cdkSetupCmd = &cobra.Command{
	Use:   "setup-cdk",
	Short: "Configures CDK v3 on the host.",
	Long:  `Configures CDK v3 on the host.`,
	Run:   runCdkSetup,
	// Make sure we skip the default persistent pre-run, since this creates already directories in $MINISHIFT_HOME
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// noop
	},
}

func init() {
	cdkSetupCmd.Flags().StringVar(&homeDir, homeDirFlag, constants.Minipath, "Sets the Minishift home directory.")
	cdkSetupCmd.Flags().StringVar(&defaultDriver, defaultDriverFlag, constants.DefaultVMDriver, "Sets the default VM driver.")
	cdkSetupCmd.Flags().BoolVar(&forceInstall, forceInstallFlag, false, "Forces the deletion of the existing Minishift install, if it exists.")
	RootCmd.AddCommand(cdkSetupCmd)
}

func runCdkSetup(cmd *cobra.Command, args []string) {
	SetupCDK()
	fmt.Println("CDK 3 setup complete.")
}

func isDefaultProfile() bool {
	return constants.ProfileName == constants.DefaultProfileName
}

func SetupCDK() {
	// When profile name is not default we need to fix the homedir
	if !isDefaultProfile() {
		cdkMarkerForDefaultProfile := filepath.Join(constants.GetMinishiftHomeDir(), CDKMarker)
		if !cdkMarkerFileExists(cdkMarkerForDefaultProfile) {
			SetupCDKMessage()
		}
		homeDir = constants.GetProfileHomeDir(constants.ProfileName)
	}
	constants.Minipath = homeDir
	fmt.Println(fmt.Sprintf("Setting up CDK 3 on host using '%s' as Minishift's home directory", constants.Minipath))

	ensureNoExistingConfigurationExists(os.Stdout, os.Stdin, homeDir)

	// Initialize the instance directory structure
	state.InstanceDirs = state.GetMinishiftDirsStructure(homeDir)

	isoCachePath, ocCachePath, configDir := createDirectories(homeDir)

	// Don't unpack ISO and OC for non-default profile
	if isDefaultProfile() {
		unpackIso(isoCachePath)
		unpackOc(ocCachePath)
	} else {
		// Use global cached ISO for non-default profile
		state.InstanceDirs.IsoCache = filepath.Join(constants.GetMinishiftHomeDir(), "cache", "iso")
		isoCachePath = filepath.Join(state.InstanceDirs.IsoCache, isoName)
	}

	createConfig(configDir, isoCachePath)
	createCDKMarker(homeDir)

	addOnManager := addon.GetAddOnManager()
	unpackAddons(addOnManager.BaseDir())
	enableDefaultAddon()
}

func createDirectories(baseDir string) (string, string, string) {
	var (
		isoCachePath string
		ocCachePath  string
	)
	// need to rebuild the directories manually here, since OcCachePath is already initialised. Improving this requires some
	// refactoring upstream (HF)
	configDir := filepath.Join(baseDir, "config")
	err := os.MkdirAll(configDir, 0777)
	check(err)

	cacheDir := filepath.Join(baseDir, "cache")
	err = os.MkdirAll(cacheDir, 0777)
	check(err)

	// Only create ISO and OC cache directories for default profile
	if isDefaultProfile() {
		isoCacheDir := filepath.Join(cacheDir, "iso")
		err = os.MkdirAll(isoCacheDir, 0777)
		check(err)
		isoCachePath = filepath.Join(isoCacheDir, isoName)

		ocCacheDir := filepath.Join(cacheDir, "oc", version.GetOpenShiftVersion(), runtime.GOOS)
		err = os.MkdirAll(ocCacheDir, 0777)
		check(err)
		ocCachePath = filepath.Join(ocCacheDir, constants.OC_BINARY_NAME)
	}

	addonsDir := filepath.Join(baseDir, "addons")
	err = os.MkdirAll(addonsDir, 0777)
	check(err)

	return isoCachePath, ocCachePath, configDir
}

func unpackOc(ocCachePath string) {
	data, err := bindata.Asset(runtime.GOOS + "/" + constants.OC_BINARY_NAME)
	check(err)

	fmt.Println(fmt.Sprintf("Copying %s to '%s'", constants.OC_BINARY_NAME, ocCachePath))
	err = ioutil.WriteFile(ocCachePath, []byte(data), 0774)
	check(err)
}

func unpackIso(iso string) {
	data, err := bindata.Asset("iso/" + isoName)
	check(err)

	fmt.Println(fmt.Sprintf("Copying %s to '%s'", isoName, iso))
	err = ioutil.WriteFile(iso, []byte(data), 0664)
	check(err)
}

func createConfig(configDir string, iso string) {
	config := make(map[string]string)
	// on Windows we might have to convert backward slashes to forward slashes
	iso = strings.Replace(iso, "\\", "/", -1)
	config["iso-url"] = "file://" + iso
	config["memory"] = CDKDefaultMemory

	if constants.ProfileName == constants.DefaultProfileName {
		config["vm-driver"] = defaultDriver
	} else {
		// If we are running createConfig for non default profile then we need to
		// get the vm-driver information from config.json of default profile
		// because Devsuite while installing CDK runs minishift setup-cdk --vm-driver <hypervisor>
		// to set the default vm-driver. So this information should not be lost when user starts a
		// new profile
		config["vm-driver"] = getVMDriverFromDefaultConfig()
	}

	configPath := filepath.Join(configDir, configFileName)
	fmt.Println(fmt.Sprintf("Creating configuration file '%s'", configPath))
	file, err := os.Create(configPath)
	check(err)

	minishiftConfig.InstanceConfig, err = minishiftConfig.NewInstanceConfig(minishiftConstants.GetInstanceConfigPath())
	check(err)

	toJson(file, config)
	err = viper.ReadInConfig()
	if err != nil {
		glog.Warningf("Error reading config file: %s", err)
	}
}

func toJson(w io.Writer, config map[string]string) {
	b, err := json.MarshalIndent(config, "", "    ")
	check(err)

	_, err = w.Write(b)
	check(err)
}

func check(e error) {
	if e != nil {
		fmt.Printf("Error setting up CDK 3 environment: %s\n", e)
		os.Exit(1)
	}
}

func ensureNoExistingConfigurationExists(w io.Writer, r io.Reader, baseDir string) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return
	}

	if !forceInstall {
		fmt.Fprintln(w, fmt.Sprintf("The MINISHIFT_HOME directory '%s' exists. Continuing will delete any existing VM and all other data in this directory. Do you want to continue? [y/N]", baseDir))

		var confirm string
		fmt.Fscanln(r, &confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Fprintln(w, "Aborting CDK setup")
			os.Exit(0)
		}
	}

	deleteExistingMachine()

	err := os.RemoveAll(baseDir)
	check(err)
}

func machineExists(api *libmachine.Client) bool {
	status, _ := cluster.GetHostStatus(api, constants.MachineName)

	if status == "Does Not Exist" {
		return false
	} else {
		return true
	}
}

func deleteExistingMachine() {
	api := libmachine.NewClient(constants.Minipath, constants.MakeMiniPath("certs"))
	defer api.Close()

	if machineExists(api) {
		if err := cluster.DeleteHost(api); err != nil {
			fmt.Println("Warning: ", err)
		}
		fmt.Println("Existing VM deleted")
	}
}

func createCDKMarker(minishiftHome string) {
	markerFileName := filepath.Join(minishiftHome, CDKMarker)
	markerFile, err := os.Create(markerFileName)
	defer markerFile.Close()
	check(err)

	fmt.Println(fmt.Sprintf("Creating marker file '%s'", markerFileName))
	tmpl := template.Must(template.New("cdkMarkerTmpl").Parse(markerTmpl))
	err = tmpl.Execute(markerFile, version.GetCDKVersion())
	check(err)
}

func unpackAddons(dir string) {
	for _, asset := range defaultAddons {
		err := addonbindata.RestoreAssets(dir, asset)
		check(err)
	}
	fmt.Println("Default add-ons", strings.Join(defaultAddons, ", "), "installed")
}

func enableDefaultAddon() {
	for _, addonName := range defaultEnabledAddons {
		addOnManager := addon.GetAddOnManager()
		enableAddon(addOnManager, addonName, 0)
	}
	fmt.Println("Default add-ons", strings.Join(defaultEnabledAddons, ", "), "enabled")
}

func enableAddon(addOnManager *manager.AddOnManager, addonName string, priority int) {
	addOnConfig, err := addOnManager.Enable(addonName, priority)
	check(err)

	minishiftConfig.InstanceConfig.AddonConfig[addOnConfig.Name] = addOnConfig
	if err := minishiftConfig.InstanceConfig.Write(); err != nil {
		atexit.ExitWithMessage(1, fmt.Sprintf("Error writing addon config data: %v", err))
	}
}

func SetupCDKMessage() {
	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	fmt.Println(fmt.Sprintf("You need to run 'minishift%s setup-cdk' first to install required CDK components.", extension))
	os.Exit(0)
}

// getVMDriverFromDefaultConfig reads the config of default profile and returns the vm-driver value
func getVMDriverFromDefaultConfig() string {
	// Check if config.json exists for default profile.
	defaultConfigFile := filepath.Join(constants.GetMinishiftHomeDir(), "config", "config.json")
	if _, err := os.Stat(defaultConfigFile); os.IsNotExist(err) {
		SetupCDKMessage()
	}

	f, err := os.Open(defaultConfigFile)
	if err != nil {
		if glog.V(2) {
			fmt.Println(fmt.Sprintf("Cannot open file %s: %s", constants.ConfigFile, err))
		}
		check(err)
	}

	m, err := decode(f)
	if err != nil {
		if glog.V(2) {
			fmt.Println(fmt.Sprintf("Cannot decode config %s: %s", defaultConfigFile, err))
		}
		check(err)
	}
	return fmt.Sprintf("%s", m["vm-driver"])
}

func decode(r io.Reader) (configCmd.MinishiftConfig, error) {
	var data configCmd.MinishiftConfig
	err := json.NewDecoder(r).Decode(&data)
	return data, err
}

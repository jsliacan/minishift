/*
Copyright (C) 2017 Red Hat, Inc.

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

package clusterup

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/provision"
	"github.com/golang/glog"

	configCmd "github.com/minishift/minishift/cmd/minishift/cmd/config"
	"github.com/minishift/minishift/pkg/minikube/kubeconfig"
	"github.com/minishift/minishift/pkg/minishift/addon/command"
	"github.com/minishift/minishift/pkg/minishift/addon/manager"
	minishiftConfig "github.com/minishift/minishift/pkg/minishift/config"
	minishiftConstants "github.com/minishift/minishift/pkg/minishift/constants"
	"github.com/minishift/minishift/pkg/minishift/oc"
	"regexp"

	"github.com/minishift/minishift/pkg/minishift/docker"
	"github.com/minishift/minishift/pkg/util"
	minishiftStrings "github.com/minishift/minishift/pkg/util/strings"
)

const (
	ipKey            = "ip"
	routingSuffixKey = "routing-suffix"
	user             = "user"
	envPrefix        = "env."
)

type ClusterUpConfig struct {
	OpenShiftVersion     string
	MachineName          string
	Ip                   string
	Port                 int
	RoutingSuffix        string
	User                 string
	Project              string
	KubeConfigPath       string
	OcPath               string
	AddonEnv             []string
	PublicHostname       string
	SSHCommander         provision.SSHCommander
	OcBinaryPathInsideVM string
	SshUser              string
}

// ClusterUp execute oc binary in order to run 'cluster up'
func ClusterUp(config *ClusterUpConfig, clusterUpParams map[string]string) (string, error) {
	cmdArgs := []string{"cluster", "up"}
	// Deal with extra flags (remove from cluster up params)
	var extraFlags string
	if val, ok := clusterUpParams[configCmd.ExtraClusterUpFlags.Name]; ok {
		extraFlags = val
		delete(clusterUpParams, configCmd.ExtraClusterUpFlags.Name)
	}

	for key, value := range clusterUpParams {
		cmdArgs = append(cmdArgs, "--"+key)
		cmdArgs = append(cmdArgs, value)
	}

	if minishiftConfig.EnableExperimental {
		// Deal with extra flags (add to command arguments)
		if len(extraFlags) > 0 {
			fmt.Println("-- Extra 'oc' cluster up flags (experimental) ... ")
			fmt.Printf("   '%s'\n", extraFlags)
			extraFlagFields := strings.Fields(extraFlags)
			for _, extraFlag := range extraFlagFields {
				cmdArgs = append(cmdArgs, extraFlag)
			}
		}
	}

	if glog.V(2) {
		fmt.Printf("-- Running 'oc' with: '%s'\n", strings.Join(cmdArgs, " "))
	}
	cmd := fmt.Sprintf("%s %s", config.OcBinaryPathInsideVM, strings.Join(cmdArgs, " "))
	out, err := config.SSHCommander.SSHCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("Error starting the cluster. %v", err)
	}
	return out, nil
}

// AddComponent add a component to running Openshift cluster
func AddComponent(sshCommander provision.SSHCommander, ocBinaryPathInsideVM string, basedir string, componentName string, imageToUse string) (string, error) {
	cmdArgs := []string{"cluster", "add", fmt.Sprintf("--base-dir=%s", basedir), fmt.Sprintf("--image=%s", imageToUse), componentName}
	if glog.V(2) {
		fmt.Printf("-- Running 'oc' with: '%s'\n", strings.Join(cmdArgs, " "))
	}
	cmd := fmt.Sprintf("%s %s", ocBinaryPathInsideVM, strings.Join(cmdArgs, " "))
	out, err := sshCommander.SSHCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("Error adding the component: %v", err)
	}
	return out, nil
}

// PostClusterUp runs the Minishift specific provisioning after 'cluster up' has run
func PostClusterUp(clusterUpConfig *ClusterUpConfig, sshCommander provision.SSHCommander, addOnManager *manager.AddOnManager, runner util.Runner) error {
	err := kubeconfig.CacheSystemAdminEntries(clusterUpConfig.KubeConfigPath, clusterUpConfig.OcBinaryPathInsideVM, sshCommander)
	if err != nil {
		return err
	}

	ocRunner, err := oc.NewOcRunner(clusterUpConfig.OcPath, clusterUpConfig.KubeConfigPath)
	if err != nil {
		return err
	}

	err = ocRunner.AddSudoerRoleForUser(clusterUpConfig.User)
	if err != nil {
		return err
	}

	err = ocRunner.AddSystemAdminEntryToKubeConfig(clusterUpConfig.OcPath)
	if err != nil {
		return err
	}

	err = ocRunner.AddCliContext(clusterUpConfig.MachineName, clusterUpConfig.Ip, clusterUpConfig.User, clusterUpConfig.Project, runner, clusterUpConfig.OcPath)
	if err != nil {
		return err
	}

	err = applyAddOns(addOnManager, clusterUpConfig.Ip, clusterUpConfig.RoutingSuffix, clusterUpConfig.SshUser, clusterUpConfig.AddonEnv, ocRunner, sshCommander)
	if err != nil {
		return err
	}

	return nil
}

// EnsureHostDirectoriesExist ensures that the specified directories exist on the VM and creates them if not.
func EnsureHostDirectoriesExist(host *host.Host, dirs []string) error {
	cmd := fmt.Sprintf("sudo install -d -o %s -g %s -m 755 %s", host.Driver.GetSSHUsername(), host.Driver.GetSSHUsername(), strings.Join(dirs, " "))
	_, err := host.RunSSHCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

// GetExecutionContext creates an ExecutionContext used for variable interpolation during add-on application.
// The context contains variables to interpolate during add-on execution, as well as the means to communicate with the VM (SSHCommander) and OpenShift (OcRunner).
func GetExecutionContext(ip string, routingSuffix string, sshUser string, addOnEnv []string, ocRunner *oc.OcRunner, sshCommander provision.SSHCommander) (*command.ExecutionContext, error) {
	context, err := command.NewExecutionContext(ocRunner, sshCommander)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to initialise execution context: %s", err.Error()))
	}

	context.AddToContext(ipKey, ip)
	context.AddToContext(routingSuffixKey, routingSuffix)
	context.AddToContext(user, sshUser)

	for _, env := range addOnEnv {
		match, _ := regexp.Match(".*=.*", []byte(env))
		if !match {
			return nil, errors.New(fmt.Sprintf("Add-on interpolation variables need to be specified in the format <key>=<value>.'%s' is not.", env))
		}

		envTokens, err := minishiftStrings.SplitAndTrim(env, "=")
		if err != nil {
			return nil, err
		}

		key, value := envTokens[0], envTokens[1]
		if strings.HasPrefix(value, envPrefix) {
			if os.Getenv(strings.TrimPrefix(value, envPrefix)) != "" {
				value = os.Getenv(strings.TrimPrefix(value, envPrefix))
			} else {
				continue
			}
		}

		context.AddToContext(key, value)
	}

	return context, nil
}

func CopyOcBinaryFromImageToVM(dockerCommander docker.DockerCommander, image string, pathInVm string) error {
	if _, err := dockerCommander.Create("--name tmp", image); err != nil {
		return err
	}
	if err := dockerCommander.Cp(minishiftConstants.OpenshiftOcExec, "tmp", pathInVm); err != nil {
		return err
	}
	if _, err := dockerCommander.Stop("tmp"); err != nil {
		return err
	}
	if err := dockerCommander.Rm("tmp"); err != nil {
		return err
	}
	return nil
}

func applyAddOns(addOnManager *manager.AddOnManager, ip string, routingSuffix string, sshUser string, addonEnv []string, ocRunner *oc.OcRunner, sshCommander provision.SSHCommander) error {
	context, err := GetExecutionContext(ip, routingSuffix, sshUser, addonEnv, ocRunner, sshCommander)
	if err != nil {
		return err
	}

	err = addOnManager.Apply(context)
	if err != nil {
		return err
	}

	return nil
}

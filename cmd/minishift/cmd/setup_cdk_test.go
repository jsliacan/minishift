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
	"bytes"
	"github.com/minishift/minishift/pkg/minikube/constants"
	"github.com/minishift/minishift/pkg/version"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
)

func Test_Ensure_No_Existing_Configuration_Exists_Aborts(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-cdk-setup-test-")
	if err != nil {
		t.Error()
	}
	defer os.RemoveAll(testDir)

	_, w, _ := os.Pipe()
	defer w.Close()

	answerReader := strings.NewReader("n")
	ensureNoExistingConfigurationExists(w, answerReader, testDir)

	t.Error("The program should have terminated")
}

func Test_Ensure_No_Existing_Configuration_Proceeds(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-cdk-setup-test-")
	if err != nil {
		t.Error()
	}
	defer os.RemoveAll(testDir)

	r, w, _ := os.Pipe()

	answerReader := strings.NewReader("y")
	ensureNoExistingConfigurationExists(w, answerReader, testDir)
	w.Close()

	s := getConsoleOutputForResponse(r)
	if strings.Contains(s, "Aborting CDK setup") {
		t.Error("CDK setup should not have aborted")
	}

	if pathExists(t, testDir) {
		t.Error(testDir + "should have been deleted.")
	}
}

func Test_Install_Into_Non_Existent_Directory_Successful(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-cdk-setup-test-")
	if err != nil {
		t.Error()
	}
	defer os.RemoveAll(testDir)

	homeDir = path.Join(testDir, "foo")
	runCdkSetup(nil, nil)

	verifyCdkSetup(t, homeDir)
}

func Test_Install_Into_Existent_Directory_With_Force_Flag_Successful(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-cdk-setup-test-")
	if err != nil {
		t.Error()
	}
	defer os.RemoveAll(testDir)

	homeDir = path.Join(testDir)
	forceInstall = true

	runCdkSetup(nil, nil)

	verifyCdkSetup(t, homeDir)
}

func Test_Create_CDK_Marker(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-cdk-setup-test-")
	if err != nil {
		t.Error()
	}
	defer os.RemoveAll(testDir)

	homeDir = path.Join(testDir)
	createCDKMarker(homeDir)

	// Test Marker creation
	markerPath := constants.MakeMiniPath(CDKMarker)
	if !cdkMarkerFileExists(markerPath) {
		t.Error(markerPath + "should have been created.")
	}

	// Test Marker content
	bytes, err := ioutil.ReadFile(markerPath)
	if err != nil {
		t.Error(err)
	}
	content := string(bytes)
	if !strings.Contains(content, version.GetCDKVersion()) {
		t.Fatalf("Marker file should contain cdk.version=%s", version.GetCDKVersion())
	}
}

func getConsoleOutputForResponse(r io.Reader) string {
	c := make(chan string)
	// copy in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		c <- buf.String()
	}()
	out := <-c
	return out
}

func pathExists(t *testing.T, path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	} else {
		t.Fatalf("Unexpected error: %v", err)
		return false
	}
}

func verifyCdkSetup(t *testing.T, homeDir string) {
	expectedPaths := [3]string{
		path.Join(homeDir, "cache", "iso", isoName),
		path.Join(homeDir, "cache", "oc", version.GetOpenShiftVersion(), constants.OC_BINARY_NAME, runtime.GOOS),
		path.Join(homeDir, "config", configFileName),
	}

	for _, v := range expectedPaths {
		if !pathExists(t, v) {
			t.Error(v + " should have been created.")
		}
	}
}

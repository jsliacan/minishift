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

package ocpdownload

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	minitesting "github.com/minishift/minishift/pkg/testing"
	minishiftos "github.com/minishift/minishift/pkg/util/os"
)

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
	err        error
	release    *github.RepositoryRelease
	resp       *github.Response
)

var testVersion = "v3.7.9"
var assetSet = []struct {
	os         minishiftos.OS
	binaryName string
	version    string
}{
	{minishiftos.LINUX, "oc", testVersion},
	{minishiftos.DARWIN, "oc", testVersion},
	{minishiftos.WINDOWS, "oc.exe", testVersion},
}

func TestDownloadOc(t *testing.T) {
	mockTransport := minitesting.NewMockRoundTripper()
	addMockResponses(mockTransport)

	client := http.DefaultClient
	client.Transport = mockTransport

	defer minitesting.ResetDefaultRoundTripper()

	testDir, err := ioutil.TempDir("", "minishift-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	for _, testAsset := range assetSet {
		err = DownloadOcpBinary(testAsset.version, testAsset.binaryName, testAsset.os, testDir)
		if err != nil {
			t.Fatal(err)
		}

		expectedBinaryPath := filepath.Join(testDir, "oc")
		if testAsset.os == minishiftos.WINDOWS {
			expectedBinaryPath += ".exe"
		}
		fileInfo, err := os.Lstat(expectedBinaryPath)
		if err != nil {
			t.Fatal(err)
		}

		if runtime.GOOS != "windows" {
			expectedFilePermissions := "-rwxrwxrwx"
			if fileInfo.Mode().String() != expectedFilePermissions {
				t.Fatalf("Wrong file permisisons. Expected %s. Got %s", expectedFilePermissions, fileInfo.Mode().String())
			}
		}

		err = os.Remove(expectedBinaryPath)
		if err != nil {
			t.Fatalf("Unable to delete %s", expectedBinaryPath)
		}
	}
}

func TestInvalidVersion(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	dummyVersion := "foo"
	err = DownloadOcpBinary(dummyVersion, "oc", minishiftos.WINDOWS, testDir)
	if err == nil {
		t.Fatal("There should have been an error")
	}

	expectedAssetURL := fmt.Sprintf("%sfoo/windows/oc.zip", OpenShiftMirrorURL)
	expectedErrorMessage := fmt.Sprintf("%s url is inaccessible", expectedAssetURL)
	if !strings.HasPrefix(err.Error(), expectedErrorMessage) {
		t.Fatalf("Expected error: '%s'. Got: '%s'\n", expectedErrorMessage, err.Error())
	}
}

func addMockResponses(mockTransport *minitesting.MockRoundTripper) {
	testDataDir := filepath.Join(basepath, "..", "..", "..", "test", "testdata")

	url := "https://mirror.openshift.com/pub/openshift-v3/clients/3.7.9/linux/oc.tar.gz$"
	mockTransport.RegisterResponse(url, &minitesting.CannedResponse{
		ResponseType: minitesting.SERVE_FILE,
		Response:     filepath.Join(testDataDir, "oc-3.7.9-linux.tar.gz"),
		ContentType:  minitesting.OCTET_STREAM,
	})

	url = "https://mirror.openshift.com/pub/openshift-v3/clients/3.7.9/macosx/oc.tar.gz$"
	mockTransport.RegisterResponse(url, &minitesting.CannedResponse{
		ResponseType: minitesting.SERVE_FILE,
		Response:     filepath.Join(testDataDir, "oc-3.7.9-darwin.tar.gz"),
		ContentType:  minitesting.OCTET_STREAM,
	})

	url = "https://mirror.openshift.com/pub/openshift-v3/clients/3.7.9/windows/oc.zip$"
	mockTransport.RegisterResponse(url, &minitesting.CannedResponse{
		ResponseType: minitesting.SERVE_FILE,
		Response:     filepath.Join(testDataDir, "oc-3.7.9-windows.zip"),
		ContentType:  minitesting.OCTET_STREAM,
	})
}

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

package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/minishift/minishift/pkg/minikube/constants"
	minitesting "github.com/minishift/minishift/pkg/testing"
	"github.com/stretchr/testify/assert"
	"net/http"
	"runtime"
)

var (
	testDir string
	testOc  Oc
)

func TestIsCached(t *testing.T) {
	setUp(t)
	defer os.RemoveAll(testDir)

	ocDir := filepath.Join(testDir, "cache", "oc", "v3.7.9", runtime.GOOS)
	os.MkdirAll(ocDir, os.ModePerm)

	assert.False(t, testOc.isCached())

	content := []byte("foo")

	err := ioutil.WriteFile(filepath.Join(ocDir, constants.OC_BINARY_NAME), content, os.ModePerm)
	assert.NoError(t, err, "Error writing to file")

	assert.True(t, testOc.isCached())
}

func TestCacheOc(t *testing.T) {
	setUp(t)
	defer os.RemoveAll(testDir) // clean up

	mockTransport := minitesting.NewMockRoundTripper()
	addMockResponses(mockTransport)

	client := http.DefaultClient
	client.Transport = mockTransport

	defer minitesting.ResetDefaultRoundTripper()

	ocDir := filepath.Join(testDir, "cache", "oc", "v3.7.9")
	os.MkdirAll(ocDir, os.ModePerm)

	err := testOc.cacheOc()
	assert.NoError(t, err, "Error caching oc")
}

func setUp(t *testing.T) {
	var err error
	testDir, err = ioutil.TempDir("", "minishift-test-")
	if err != nil {
		t.Error()
	}
	testOc = Oc{"v3.7.9", filepath.Join(testDir, "cache")}
}

func addMockResponses(mockTransport *minitesting.MockRoundTripper) {
	testDataDir := filepath.Join("..", "..", "..", "test", "testdata")

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

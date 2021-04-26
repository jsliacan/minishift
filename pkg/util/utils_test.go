/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	minitesting "github.com/minishift/minishift/pkg/testing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
)

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// Returns a function that will return n errors, then return successfully forever.
func errorGenerator(n int, retryable bool) func() error {
	errorCount := 0
	return func() (err error) {
		if errorCount < n {
			errorCount += 1
			e := errors.New("Error!")
			if retryable {
				return &RetriableError{Err: e}
			} else {
				return e
			}

		}

		return nil
	}
}

func TestErrorGenerator(t *testing.T) {
	errors := 3
	f := errorGenerator(errors, false)
	for i := 0; i < errors-1; i++ {
		err := f()
		assert.Error(t, err, fmt.Sprintf("Error should have been thrown at iteration %v", i))
	}
	err := f()
	assert.Error(t, err)
}

func TestRetry(t *testing.T) {
	f := errorGenerator(4, true)
	err := Retry(5, f) //err != nil {
	assert.NoError(t, err, "Error should not have been raised by retry")

	f = errorGenerator(5, true)
	err = Retry(4, f)
	assert.Error(t, err, "Error should have been raised by retry")
}

func TestMultiError(t *testing.T) {
	m := MultiError{}

	m.Collect(errors.New("Error 1"))
	m.Collect(errors.New("Error 2"))

	err := m.ToError()
	expected := `Error 1
Error 2`

	assert.Equal(t, expected, err.Error())

	m = MultiError{}
	err = m.ToError()
	assert.NoError(t, err)
}

var durationTests = []struct {
	in   time.Duration
	want string
}{
	{10*time.Second + 555*time.Millisecond, "10.6s"},
	{10*time.Second + 555*time.Millisecond, "10.6s"},
	{10*time.Second + 500*time.Millisecond, "10.5s"},
	{10*time.Second + 499*time.Millisecond, "10.5s"},
	{9*time.Second + 401*time.Millisecond, "9.4s"},
	{9*time.Second + 456*time.Millisecond, "9.46s"},
	{9*time.Second + 445*time.Millisecond, "9.45s"},
	{1 * time.Second, "1s"},
	{859*time.Millisecond + 445*time.Microsecond, "859.4ms"},
	{859*time.Millisecond + 460*time.Microsecond, "859.5ms"},
	{859*time.Microsecond + 100*time.Nanosecond, "900µs"},
	{45 * time.Nanosecond, "45ns"},
}

func TestFriendlyDuration(t *testing.T) {
	for _, tt := range durationTests {
		got := FriendlyDuration(tt.in)
		expected, _ := time.ParseDuration(tt.want)
		assert.Equal(t, expected, got)
	}
}

func Test_command_executes_successfully_with_absolute_path(t *testing.T) {
	currentDir, err := os.Getwd()
	assert.NoError(t, err, "Error getting working directory")

	dummyBinaryPath := filepath.Join(currentDir, "..", "..", "test", "testdata")

	switch os := runtime.GOOS; os {
	case "darwin":
		dummyBinaryPath = filepath.Join(dummyBinaryPath, "dummybinary_darwin")
	case "linux":
		dummyBinaryPath = filepath.Join(dummyBinaryPath, "dummybinary_linux")
	case "windows":
		dummyBinaryPath = filepath.Join(dummyBinaryPath, "dummybinary_windows.exe")
	default:
		assert.FailNow(t, "Unpexpected OS")
	}

	testData := []struct {
		command string
		exists  bool
	}{
		{filepath.Join(currentDir, "..", "..", "test", "testdata", "blahh"), false},
		{dummyBinaryPath, true},
	}
	for _, v := range testData {
		got := CommandExecutesSuccessfully(v.command)
		assert.Equal(t, v.exists, got, "while executing %s", dummyBinaryPath)
	}
}

func Test_command_executes_successfully_with_command_lookup(t *testing.T) {
	currentDir, err := os.Getwd()
	assert.NoError(t, err, "Error getting working directory")

	dummyBinaryPath := filepath.Join(currentDir, "..", "..", "test", "testdata")
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dummyBinaryPath+string(os.PathListSeparator)+origPath)
	defer func() {
		os.Setenv("PATH", origPath)
	}()

	var cmd string
	switch os := runtime.GOOS; os {
	case "darwin":
		cmd = "dummybinary_darwin"
	case "linux":
		cmd = "dummybinary_linux"
	case "windows":
		cmd = "dummybinary_windows.exe"
	default:
		assert.FailNow(t, "Unpexpected OS")
	}

	success := CommandExecutesSuccessfully(cmd)
	assert.True(t, success)
}

func Test_command_executes_unsuccessfully_with_command_lookup(t *testing.T) {
	var cmd string
	switch os := runtime.GOOS; os {
	case "darwin":
		cmd = "dummybinary_darwin"
	case "linux":
		cmd = "dummybinary_linux"
	case "windows":
		cmd = "dummybinary_windows.exe"
	default:
		assert.FailNow(t, "Unpexpected OS")
	}

	success := CommandExecutesSuccessfully(cmd)
	assert.False(t, success)
}

func TestWritableDirectory(t *testing.T) {
	testDir, err := ioutil.TempDir("", "minishift-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	assert.True(t, IsDirectoryWritable(testDir))
}

func TestNonWritableDirectory(t *testing.T) {
	if isAppVeyorUser() ||
		IsAdministrativeUser() {
		t.Skip("Skipping as CI runs as root or administrator")
	}

	testDir, err := ioutil.TempDir("", "minishift-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	os.Chmod(testDir, 0400) // make dir read-only
	assert.False(t, IsDirectoryWritable(testDir))

}

func isAppVeyorUser() bool {
	// Checking for $env:APPVEYOR does not work !
	u, _ := user.Current()
	return strings.Contains(u.Username, "appveyor")
}

func TestValidOcpVersionAvailable(t *testing.T) {
	testVersion := "3.7.23"
	mockTransport := minitesting.NewMockRoundTripper()
	addMockResponses(mockTransport)

	client := http.DefaultClient
	client.Transport = mockTransport
	defer minitesting.ResetDefaultRoundTripper()

	assert.True(t, IsOcpVersionAvailable(testVersion))
}

func TestInValidOcpVersionAvailable(t *testing.T) {
	testVersion := "1.1.1"
	mockTransport := minitesting.NewMockRoundTripper()
	addMockResponses(mockTransport)

	client := http.DefaultClient
	client.Transport = mockTransport
	defer minitesting.ResetDefaultRoundTripper()

	assert.False(t, IsOcpVersionAvailable(testVersion))
}

func addMockResponses(mockTransport *minitesting.MockRoundTripper) {
	testDataDir := filepath.Join(basepath, "..", "..", "test", "testdata")

	url := "https://mirror.openshift.com/pub/openshift-v3/clients/$"
	mockTransport.RegisterResponse(url, &minitesting.CannedResponse{
		ResponseType: minitesting.SERVE_FILE,
		Response:     filepath.Join(testDataDir, "mirror_openshift_com.html"),
		ContentType:  minitesting.OCTET_STREAM,
	})
}

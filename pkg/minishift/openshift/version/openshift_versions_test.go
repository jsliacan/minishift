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

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPrerelease(t *testing.T) {
	var versionTestData = []struct {
		openshiftVersion string
		expectedResult   bool
	}{
		{"v3.6.0", false},
		{"v3.6.0-alpha.1", true},
		{"v3.9.0-alpha.3", true},
		{"v3.8.0-rc.1", true},
		{"v3.6.0-beta", true},
	}

	for _, versionTest := range versionTestData {
		actualResult := isPrerelease(versionTest.openshiftVersion)
		assert.Equal(t, versionTest.expectedResult, actualResult)
	}
}

func TestOpenShiftTagsByAscending(t *testing.T) {
	var testTags = []string{"v3.7.0", "v3.10.0", "v3.7.2", "v3.9.0", "v3.7.1"}

	sortTags, _ := OpenShiftTagsByAscending(testTags, "v3.7.0", "v3.9.0")
	assert.Equal(t, sortTags[len(sortTags)-1], "v3.10.0")
}

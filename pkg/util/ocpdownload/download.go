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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/minishift/minishift/pkg/util/archive"
	minishiftOS "github.com/minishift/minishift/pkg/util/os"
	"gopkg.in/cheggaaa/pb.v1"
)

const (
	OpenShiftMirrorURL = "https://mirror.openshift.com/pub/openshift-v3/clients/"
	TAR                = "tar.gz"
	ZIP                = "zip"
)

func DownloadOcpBinary(version, binaryName string, osType minishiftOS.OS, outputPath string) error {
	binaryType := strings.TrimRight(binaryName, ".exe") // Remove .exe if exists
	platform := osType
	if osType == minishiftOS.DARWIN {
		platform = "macosx"
	}

	extension := TAR
	if osType == minishiftOS.WINDOWS {
		extension = ZIP
	}

	url := fmt.Sprintf("%s%s/%s/%s.%s", OpenShiftMirrorURL, strings.TrimPrefix(version, "v"), platform, binaryType, extension)
	targetFile := fmt.Sprintf("%s.%s", binaryType, extension)

	// Create target directory and file
	tmpDir, err := ioutil.TempDir("", "minishift-asset-download-")
	if err != nil {
		return errors.New("cannot create temporary download directory")
	}
	defer os.RemoveAll(tmpDir)

	// Create a tmp directory for the asset
	assetTmpFile := filepath.Join(tmpDir, targetFile)
	// Download the actual file
	fmt.Println(fmt.Sprintf("-- Downloading OpenShift Container Platform client binary '%s'", binaryType))
	if _, err := download(url, assetTmpFile); err != nil {
		return err
	}

	binaryPath := ""
	switch {
	case strings.HasSuffix(assetTmpFile, TAR):
		// unzip
		tarFile := assetTmpFile[:len(assetTmpFile)-3]
		err = archive.Ungzip(assetTmpFile, tarFile)
		if err != nil {
			return errors.New("cannot ungzip")
		}

		// untar
		err = archive.Untar(tarFile, tmpDir)
		if err != nil {
			return errors.New("cannot untar")
		}

		content, err := listDirExcluding(tmpDir, ".*.tar.*")
		if err != nil {
			return errors.New("cannot list content of")
		}
		if len(content) > 1 {
			return errors.New(fmt.Sprintf("Unexpected number of files in tmp directory: %s", content))
		}

		binaryPath = tmpDir
	case strings.HasSuffix(assetTmpFile, ZIP):
		//contentDir := assetTmpFile[:len(assetTmpFile)-4]
		err = archive.Unzip(assetTmpFile, tmpDir)
		if err != nil {
			return errors.New("cannot unzip")
		}
		binaryPath = tmpDir
	}

	binaryPath = filepath.Join(binaryPath, binaryName)

	// Copy the requested asset into its final destination
	err = os.MkdirAll(outputPath, 0755)
	if err != nil && !os.IsExist(err) {
		return errors.New("cannot create the target directory")
	}

	finalBinaryPath := filepath.Join(outputPath, binaryName)
	err = copy(binaryPath, finalBinaryPath)
	if err != nil {
		return err
	}

	err = os.Chmod(finalBinaryPath, 0777)
	if err != nil {
		return fmt.Errorf("cannot make executable: %s", err.Error())
	}

	return nil
}

func listDirExcluding(dir string, excludeRegexp string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, f := range files {
		matched, err := regexp.MatchString(excludeRegexp, f.Name())
		if err != nil {
			return nil, err
		}

		if !matched {
			result = append(result, f.Name())
		}

	}

	return result, nil
}

func copy(src, dest string) error {
	srcFile, err := os.Open(src)
	defer srcFile.Close()
	if err != nil {
		return fmt.Errorf("cannot open src file: %s", src)
	}

	destFile, err := os.Create(dest)
	defer destFile.Close()
	if err != nil {
		return fmt.Errorf("cannot create dst file: %s", dest)
	}

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("cannot copy: %s", err.Error())
	}

	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("cannot copy: %s", err.Error())
	}

	return nil
}

func download(url string, filename string) (bool, error) {
	httpResp, err := http.Get(url)
	if err != nil {
		return false, errors.New(fmt.Sprintf("cannot download OpenShift container platform client binary: %s", err.Error()))
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode == 404 {
		return false, fmt.Errorf("%s url is inaccessible", url)
	}

	asset := httpResp.Body
	if httpResp.ContentLength > 0 {
		bar := pb.New64(httpResp.ContentLength).SetUnits(pb.U_BYTES)
		bar.Start()
		asset = bar.NewProxyReader(asset)
		defer func() {
			<-time.After(bar.RefreshRate)
			fmt.Println()
		}()
	}

	out, err := os.Create(filename)
	defer out.Close()
	if err != nil {
		return false, fmt.Errorf("not able to create file as '%s': %s", url, err.Error())
	}
	_, err = io.Copy(out, asset)
	if err != nil {
		return false, fmt.Errorf("not able to copy file to '%s': %s", filename, err.Error())
	}

	return true, nil
}

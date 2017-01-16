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
	"github.com/minishift/minishift/pkg/util/ocpdownload"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type RetriableError struct {
	Err error
}

func (r RetriableError) Error() string { return "Temporary Error: " + r.Err.Error() }

// Until endlessly loops the provided function until a message is received on the done channel.
// The function will wait the duration provided in sleep between function calls. Errors will be sent on provider Writer.
func Until(fn func() error, w io.Writer, name string, sleep time.Duration, done <-chan struct{}) {
	var exitErr error
	for {
		select {
		case <-done:
			return
		default:
			exitErr = fn()
			if exitErr == nil {
				fmt.Fprintf(w, Pad("%s: Exited with no errors.\n"), name)
			} else {
				fmt.Fprintf(w, Pad("%s: Exit with error: %v"), name, exitErr)
			}

			// wait provided duration before trying again
			time.Sleep(sleep)
		}
	}
}

func Pad(str string) string {
	return fmt.Sprintf("\n%s\n", str)
}

func Retry(attempts int, callback func() error) (err error) {
	return RetryAfter(attempts, callback, 0)
}

func RetryAfter(attempts int, callback func() error, d time.Duration) (err error) {
	m := MultiError{}
	for i := 0; i < attempts; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		m.Collect(err)
		if _, ok := err.(*RetriableError); !ok {
			return m.ToError()
		}
		time.Sleep(d)
	}
	return m.ToError()
}

type MultiError struct {
	Errors []error
}

func (m *MultiError) Collect(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m MultiError) ToError() error {
	if len(m.Errors) == 0 {
		return nil
	}

	errStrings := []string{}
	for _, err := range m.Errors {
		errStrings = append(errStrings, err.Error())
	}
	return fmt.Errorf(strings.Join(errStrings, "\n"))
}

// TimeTrack is used to time the execution of a method. It is passed the start time as well as a output writer for the timing.
// The usage of TimeTrack is in combination with defer like so:
//
//    defer TimeTrack(time.Now(), os.Stdout)
func TimeTrack(start time.Time, w io.Writer, friendly bool) {
	fmt.Fprintln(w, fmt.Sprintf("[%v]", TimeElapsed(start, friendly)))
}

// TimeElapsed is used to time the execution of a method.
func TimeElapsed(start time.Time, friendly bool) string {
	elapsed := time.Since(start)

	if friendly {
		elapsed = FriendlyDuration(elapsed)
	}

	return elapsed.String()
}

func FriendlyDuration(d time.Duration) time.Duration {
	if d > 10*time.Second {
		d2 := ((d + 50*time.Millisecond) / (100 * time.Millisecond)) * (100 * time.Millisecond)
		return d2
	}
	if d > time.Second {
		d2 := ((d + 5*time.Millisecond) / (10 * time.Millisecond)) * (10 * time.Millisecond)
		return d2
	}
	if d > time.Microsecond {
		d2 := ((d + 50*time.Microsecond) / (100 * time.Microsecond)) * (100 * time.Microsecond)
		return d2
	}

	d2 := (d / time.Nanosecond) * (time.Nanosecond)
	return d2
}

// CommandExecutesSuccessfully returns true if the command executed based on the exit code
func CommandExecutesSuccessfully(cmd string, args ...string) bool {
	var runner Runner = &RealRunner{}
	var stdOut, stdErr io.Writer

	exitCode := runner.Run(stdOut, stdErr, cmd, args...)

	if exitCode == 0 {
		return true
	}
	return false
}

// IsDirectoryWritable checks if the directory is writable or not
// by trying creating a temporary file
// Return true if directory is writable else false
func IsDirectoryWritable(path string) bool {
	tmpFilePath := filepath.Join(path, "tmp")
	_, err := os.Create(tmpFilePath)
	if err != nil {
		return false
	}
	defer os.Remove(tmpFilePath)

	return true
}

// IsAdministrativeUser returns true when user is either root or
// Administrator
func IsAdministrativeUser() bool {
	u, _ := user.Current()
	username := strings.ToLower(u.Username)

	return u.Uid == "1" ||
		username == "root" ||
		strings.Contains(username, "administrator")
}

// getHref parses the token and return the href attribute
func getHref(t html.Token) (bool, string) {
	var (
		ok   bool
		href string
	)

	// Iterate over all of the Token's attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	return ok, href
}

// IsOcpVersionAvailable parses the OpenShift mirror url and gets a list of ocp versions
// from anchor tags(href) and matches the specified version.
// Return true if version is matched else false.
func IsOcpVersionAvailable(version string) bool {
	resp, err := http.Get(ocpdownload.OpenShiftMirrorURL)
	if err != nil {
		return false
	}
	b := resp.Body
	defer b.Close()

	z := html.NewTokenizer(b)
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the HTML document, return false since we couldn't find specified version
			return false
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			if t.Data != "a" {
				continue
			}

			// Extract the href value, if there is one
			ok, href := getHref(t)
			if !ok {
				continue
			}

			match, _ := regexp.Match(fmt.Sprintf("^%s/$", version), []byte(href))
			if match {
				return true
			}
		}
	}
}

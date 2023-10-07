//go:build ignore
// +build ignore

// The make_version program is run by go generate to compile a version stamp
// to be compiled into the goProbe and goQuery binary
// It does nothing unless $COMMIT_SHA is set, which is true only during
// the release process.
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	version := os.Getenv("COMMIT_SHA")
	if version == "" {
		return
	}
	semver := os.Getenv("SEM_VER")

	output := fmt.Sprintf(outputFormat, time.Now().In(time.UTC).Format(time.UnixDate), version, semver)

	err := os.WriteFile("git_version.go", []byte(output), 0664)
	if err != nil {
		panic(err)
	}
}

const outputFormat = `
// Code generated by 'go run make_version.go'. DO NOT EDIT.
package version
import (
    "fmt"
    "time"
)
func init() {
    var err error
    BuildTime, err = time.Parse(time.UnixDate, %[1]q)
    if err != nil {
        panic(err)
    }
    GitSHA = fmt.Sprintf("%%s", %[2]q)
    SemVer = fmt.Sprintf("%%s", %[3]q)
}
`
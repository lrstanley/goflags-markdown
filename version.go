// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package clix

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/gookit/color"
)

// Module represents a module.
type Module struct {
	Path    string  `json:"path,omitempty"`     // module path
	Version string  `json:"version,omitempty"`  // module version
	Sum     string  `json:"sum,omitempty"`      // checksum
	Replace *Module `json:"replaces,omitempty"` // replaced by this module
}

func (m Module) String() string {
	if m.Replace != nil {
		m = *m.Replace
	}

	return fmt.Sprintf("%s :: %s :: %s", m.Sum, m.Path, m.Version)
}

// BuildSetting describes a setting that may be used to understand how the
// binary was built. For example, VCS commit and dirty status is stored here.
type BuildSetting struct {
	// Key and Value describe the build setting.
	// Key must not contain an equals sign, space, tab, or newline.
	// Value must not contain newlines ('\n').
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s BuildSetting) String() string {
	return fmt.Sprintf("%s: %s", s.Key, s.Value)
}

// VersionOptions are the options used when querying and returning version information.
type VersionOptions struct {
	DisableBuildSettings bool `json:"-"` // Disable printing build settings.
	DisableDeps          bool `json:"-"` // Disable printing dependencies.
}

// Version represents the version information for the CLI.
type Version struct {
	options *VersionOptions `json:"-"`

	Application  Application    `json:"application,omitempty"`    // Application information.
	Settings     []BuildSetting `json:"build_settings,omitempty"` // Other information about the build.
	Dependencies []Module       `json:"dependencies,omitempty"`   // Module dependencies.

	Command   string `json:"command"`    // Executable name where the command was called from.
	GoVersion string `json:"go_version"` // Version of Go that produced this binary.
	OS        string `json:"os"`         // Operating system for this build.
	Arch      string `json:"arch"`       // CPU Architecture for build build.
}

// NonSensitiveVersion represents the version information for the CLI.
type NonSensitiveVersion struct {
	options *VersionOptions `json:"-"`

	Application Application `json:"application,omitempty"` // Application information.
	Command     string      `json:"command"`               // Executable name where the command was called from.
	GoVersion   string      `json:"go_version"`            // Version of Go that produced this binary.
	OS          string      `json:"os"`                    // Operating system for this build.
	Arch        string      `json:"arch"`                  // CPU Architecture for build build.
}

// NonSensitive returns a copy of Version with sensitive information removed.
func (v *Version) NonSensitive() *NonSensitiveVersion {
	return &NonSensitiveVersion{
		options:     v.options,
		Application: v.Application,
		Command:     v.Command,
		GoVersion:   v.GoVersion,
		OS:          v.OS,
		Arch:        v.Arch,
	}
}

// GetSetting returns the value of the setting with the given key, otherwise
// defaults to defaultValue.
func (v *Version) GetSetting(key, defaultValue string) string {
	if v.Settings == nil {
		return defaultValue
	}

	for _, s := range v.Settings {
		if s.Key == key {
			return s.Value
		}
	}

	return defaultValue
}

func (v *Version) stringBase() string {
	w := &bytes.Buffer{}

	fmt.Fprintf(w, "<cyan>%s</> :: <yellow>%s</>\n", v.Application.Name, v.Application.Version)
	fmt.Fprintf(w, "|  build commit :: <green>%s</>\n", v.Application.Commit)
	fmt.Fprintf(w, "|    build date :: <green>%s</>\n", v.Application.Date)
	fmt.Fprintf(w, "|    go version :: <green>%s %s/%s</>\n", v.GoVersion, v.OS, v.Arch)

	if len(v.Application.Links) > 0 {
		var longest int
		for _, l := range v.Application.Links {
			if len(l.Name) > longest {
				longest = len(l.Name)
			}
		}

		fmt.Fprintf(w, "\n<cyan>helpful links:</>\n")
		for _, l := range v.Application.Links {
			fmt.Fprintf(
				w, "|  %s%s :: <magenta>%s</>\n",
				strings.Repeat(" ", longest-len(l.Name)),
				l.Name, l.URL,
			)
		}
	}

	return w.String()
}

func (v *Version) String() string {
	w := &bytes.Buffer{}

	w.WriteString(v.stringBase())

	if v.options == nil || !v.options.DisableBuildSettings {
		var longest int
		for _, s := range v.Settings {
			if len(s.Key) > longest {
				longest = len(s.Key)
			}
		}

		fmt.Fprintf(w, "\n<cyan>build options:</>\n")
		for _, s := range v.Settings {
			fmt.Fprintf(
				w, "|  %s%s :: <magenta>%s</>\n",
				strings.Repeat(" ", longest-len(s.Key)),
				s.Key, s.Value,
			)
		}
	}

	if v.options == nil || !v.options.DisableDeps {
		fmt.Fprintf(w, "\n<cyan>dependencies:</>\n")
		for _, m := range v.Dependencies {
			if m.Replace != nil {
				m = *m.Replace
			}

			if m.Sum == "" {
				m.Sum = "unknown"
			}

			fmt.Fprintf(w, "  %47s :: <cyan>%s</> :: <yellow>%s</>\n", m.Sum, m.Path, m.Version)
		}
	}

	return color.Sprint(w.String())
}

// GetVersionInfo returns the version information for the CLI.
func GetVersionInfo(app Application, options *VersionOptions) *Version {
	v := &Version{
		options:     options,
		Application: app,
		GoVersion:   runtime.Version(),
		Command:     filepath.Base(os.Args[0]),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
	}

	build, ok := debug.ReadBuildInfo()
	if ok {
		if v.Settings == nil {
			v.Settings = make([]BuildSetting, 0, len(build.Settings))
			for _, setting := range build.Settings {
				v.Settings = append(v.Settings, BuildSetting{
					Key:   setting.Key,
					Value: setting.Value,
				})
			}
		}

		if v.Dependencies == nil {
			v.Dependencies = make([]Module, 0, len(build.Deps))
			for _, dep := range build.Deps {
				v.Dependencies = append(v.Dependencies, Module{
					Path:    dep.Path,
					Version: dep.Version,
					Sum:     dep.Sum,
				})
			}
		}

		if v.Application.Name == "" {
			v.Application.Name = build.Main.Path
		}

		if v.Application.Version == "" {
			v.Application.Version = build.Main.Version
		}

		if v.Application.Commit == "" {
			v.Application.Commit = v.GetSetting("vcs.revision", build.Main.Sum)
		}

		if v.Application.Date == "" {
			v.Application.Date = v.GetSetting("vcs.time", "unknown")
		}
	}

	if v.Application.Name == "" {
		v.Application.Name = v.Command
	}

	if v.Application.Version == "" {
		v.Application.Version = "unknown"
	}

	if v.Application.Commit == "" {
		v.Application.Commit = "unknown"
	}

	if v.Application.Date == "" {
		v.Application.Date = "unknown"
	}

	if v.Application.Description == "" {
		v.Application.Description = color.Sprint(v.stringBase())
	}

	return v
}

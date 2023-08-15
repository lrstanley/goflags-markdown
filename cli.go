// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package clix

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/apex/log"
	_ "github.com/joho/godotenv/autoload"
)

// Options allows overriding default logic.
type Options int

const (
	OptDisableLogging       Options = 1 << iota // Disable logging initialization.
	OptDisableVersion                           // Disable version printing (must handle manually).
	OptDisableDeps                              // Disable dependency printing in version output.
	OptDisableBuildSettings                     // Disable build info printing in version output.
	OptDisableGlobalLogger                      // Disable setting the global logger for apex/log.
	OptSubcommandsOptional                      // Subcommands are optional.
)

type Application struct {
	Name        string `json:"name"`          // Application name.
	Description string `json:"description"`   // Application description.
	Version     string `json:"build_version"` // Build version.
	Commit      string `json:"build_commit"`  // VCS commit SHA.
	Date        string `json:"build_date"`    // VCS commit date.

	Links []Link `json:"links,omitempty"` // Links to the project's website, support, issues, security, etc.
}

// CLI is the main construct for clix. Do not manually set any fields until
// you've called Parse(). Initialize a new CLI like so:
//
//	var (
//		logger   log.Interface
//		cli    = &clix.CLI[Flags]{} // Where Flags is a user-provided type (struct).
//	)
//
//	type Flags struct {
//		SomeFlag string `long:"some-flag" description:"some flag"`
//	}
//
//	// [...]
//	cli.Parse(clix.OptDisableGlobalLogger|clix.OptDisableBuildSettings)
//	logger = cli.Logger
//
// Additional notes:
// * Use cli.Logger as a apex/log log.Interface (as shown above).
// * Use cli.Args to get the remaining arguments provided to the program.
type CLI[T any] struct {
	options Options  `kong:"-"`
	version *Version `kong:"-"`

	// Flags are the user-provided flags.
	Flags *T `embed:""`

	// Context is the context returned by kong after initial parsing.
	Context *kong.Context `kong:"-"`

	// Application should contain basic information about your application.
	Application Application `kong:"-"`

	// VersionOptions are used to configure how version information should be
	// represented.
	VersionOptions *VersionOptions `kong:"-"`

	// Version can be used to print the version information to console. Use
	// NO_COLOR or FORCE_COLOR to change coloring.
	VersionFlag struct {
		Enabled     bool `short:"v" env:"-" name:"version" help:"prints version information and exits"`
		EnabledJSON bool `name:"version-json" env:"-" help:"prints version information in JSON format and exits"`
	} `embed:""`

	// Debug can be used to enable/disable debugging as a global flag. Also
	// sets the log level to debug.
	Debug bool `short:"D" name:"debug" help:"enables debug mode"`

	// GenerateMarkdown can be used to generate markdown documentation for
	// the cli. clix will intercept and output the documentation to stdout.
	GenerateMarkdown bool `name:"generate-markdown" env:"-" hidden:"" help:"generate markdown documentation and write to stdout"`

	// Logger is the generated logger.
	Logger       *log.Logger  `kong:"-"`
	LoggerConfig LoggerConfig `embed:"" group:"Logging Options" prefix:"log."`
}

// GetVersion returns the version information for the CLI, which will be populated
// after parsing.
func (cli *CLI[T]) GetVersion() *Version {
	return cli.version
}

// Parse executes the go-flags parser, returns the remaining arguments, as
// well as initializes a new logger. If cli.Version is set, it will print
// the version information (unless disabled).
func (cli *CLI[T]) Parse(options ...Options) *kong.Context {
	return cli.ParseWithKongOptions(
		options,
		[]kong.Option{
			kong.Configuration(kong.JSON),
			kong.ConfigureHelp(kong.HelpOptions{
				Tree:      true,
				FlagsLast: true,
			}),
			kong.DefaultEnvars(""),
			kong.UsageOnError(),
		},
	)
}

func (cli *CLI[T]) ParseWithKongOptions(options []Options, kongOptions []kong.Option) *kong.Context {
	if cli.Flags == nil {
		cli.Flags = new(T)
	}

	cli.Set(options...)
	cli.version = GetVersionInfo(cli.Application, cli.VersionOptions)
	cli.Application = cli.version.Application // The version info can also help fill out the app info.

	kongOptions = append([]kong.Option{kong.Name(cli.Application.Name), kong.Description(cli.Application.Description)}, kongOptions...)

	cli.Context = kong.Parse(cli, kongOptions...)

	// Initialize the logger.
	if !cli.IsSet(OptDisableLogging) {
		cli.newLogger()
	}

	if (cli.VersionFlag.EnabledJSON) && !cli.IsSet(OptDisableVersion) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(cli.version); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	if (cli.VersionFlag.Enabled) && !cli.IsSet(OptDisableVersion) {
		fmt.Println(cli.version.String())
		os.Exit(1)
	}

	// if cli.GenerateMarkdown {
	// 	cli.Markdown(os.Stdout)
	// 	os.Exit(0)
	// }

	if !cli.IsSet(OptDisableLogging) {
		cli.Logger.WithFields(log.Fields{
			"name":       cli.Application.Name,
			"version":    cli.Application.Version,
			"commit":     cli.Application.Commit,
			"go_version": cli.version.GoVersion,
			"os":         cli.version.OS,
			"arch":       cli.version.Arch,
		}).Debug("logger initialized")
	}

	// if command != nil {
	// 	err := initFn()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	return command.Execute(args)
	// }

	return cli.Context
}

// IsSet returns true if the given option is set.
func (cli *CLI[T]) IsSet(options Options) bool {
	return cli.options&options != 0
}

// Set sets the given option.
func (cli *CLI[T]) Set(options ...Options) {
	for _, o := range options {
		cli.options |= o
	}
}

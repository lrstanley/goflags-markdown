// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package clix

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/gookit/color"
	flags "github.com/jessevdk/go-flags"
)

// Options allows overriding default logic.
type Options int

const (
	OptDisableLogging       Options = 1 << iota // Disable logging initialization.
	OptDisableVersion                           // Disable version printing (must handle manually).
	OptDisableDeps                              // Disable dependency printing in version output.
	OptDisableBuildSettings                     // Disable build info printing in version output.
	OptDisableGlobalLogger                      // Disable setting the global logger for apex/log.
)

// CLI is the main construct for clix. Do not manually set any fields until
// you've called Parse(). Initialize a new CLI like so:
//
//  var (
//  	logger   log.Interface
//  	cli    = &clix.CLI[Flags]{} // Where Flags is a user-provided type (struct).
//  )
//
//  type Flags struct {
//  	SomeFlag string `long:"some-flag" description:"some flag"`
//  }
//
//  // [...]
//  cli.Parse(clix.OptDisableGlobalLogger|clix.OptDisableBuildSettings)
//  logger = cli.Logger
//
// Additional notes:
// * Use cli.Logger as a apex/log log.Interface (as shown above).
// * Use cli.Args to get the remaining arguments provided to the program.
type CLI[T any] struct {
	// Flags are the user-provided flags.
	Flags *T

	// VersionInfo is the version information for the CLI. You can provide
	// a custom version of this if you already have version information.
	VersionInfo *VersionInfo[T] `json:"version_info"`

	// Links are the links to the project's website, support, issues, security,
	// etc. This will be used in help and version output if provided.
	// Links are in the format of "name=url".
	Links []Link

	// Args are the remaining arguments after parsing.
	Args []string

	// Version can be used to print the version information to console. Use
	// NO_COLOR or FORCE_COLOR to change coloring.
	Version struct {
		Enabled     bool `short:"v" long:"version" description:"prints version information and exits"`
		EnabledJSON bool `long:"version-json" description:"prints version information in JSON format and exits"`
	}

	// Debug can be used to enable/disable debugging as a global flag. Also
	// sets the log level to debug.
	Debug bool `short:"D" long:"debug" env:"DEBUG" description:"enables debug mode"`

	// GenerateMarkdown can be used to generate markdown documentation for
	// the cli. clix will intercept and output the documentation to stdout.
	GenerateMarkdown bool `long:"generate-markdown" hidden:"true" description:"generate markdown documentation and write to stdout" json:"-"`

	// Logger is the generated logger.
	Logger       *log.Logger  `json:"-"`
	LoggerConfig LoggerConfig `group:"Logging Options" namespace:"log" env-namespace:"LOG"`

	options Options `json:"-"`
}

// Parse executes the go-flags parser, returns the remaining arguments, as
// well as initializes a new logger. If cli.Version is set, it will print
// the version information (unless disabled).
func (cli *CLI[T]) Parse(options ...Options) *CLI[T] {
	if cli.Flags == nil {
		cli.Flags = new(T)
	}

	cli.Set(options...)
	cli.VersionInfo = cli.GetVersionInfo()

	parser := cli.newParser()
	parser.LongDescription = color.Sprint(cli.VersionInfo.stringBase())

	args, err := parser.Parse()
	if err != nil {
		if FlagErr, ok := err.(*flags.Error); ok && FlagErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	cli.Args = args

	// Initialize the logger.
	if !cli.IsSet(OptDisableLogging) {
		cli.newLogger()
	}

	if (cli.Version.EnabledJSON) && !cli.IsSet(OptDisableVersion) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(cli.VersionInfo); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	if (cli.Version.Enabled) && !cli.IsSet(OptDisableVersion) {
		fmt.Println(cli.VersionInfo.String())
		os.Exit(1)
	}

	if cli.GenerateMarkdown {
		cli.Markdown(os.Stdout)
		os.Exit(0)
	}

	if !cli.IsSet(OptDisableLogging) {
		cli.Logger.WithFields(log.Fields{
			"name":       cli.VersionInfo.Name,
			"version":    cli.VersionInfo.Version,
			"commit":     cli.VersionInfo.Commit,
			"go_version": cli.VersionInfo.GoVersion,
			"os":         cli.VersionInfo.OS,
			"arch":       cli.VersionInfo.Arch,
		}).Info("logger initialized")
	}

	return cli
}

func (cli *CLI[T]) newParser() (p *flags.Parser) {
	p = flags.NewParser(cli, flags.PrintErrors|flags.HelpFlag|flags.PassDoubleDash)

	p.NamespaceDelimiter = "."
	p.EnvNamespaceDelimiter = "_"

	return p
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

// Link allows you to define a link to be included in the version and usage
// output.
type Link struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GithubLinks return an opinonated set of links for the project, using
// common Github layout conventions.
func GithubLinks(repo, branch, homepage string) []Link {
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimSuffix(repo, "/")

	links := []Link{}

	if branch == "" {
		branch = "master"
	}

	if homepage != "" {
		links = append(links, Link{
			Name: "homepage",
			URL:  homepage,
		})
	}

	links = append(links, []Link{
		{Name: "github", URL: fmt.Sprintf("https://%s", repo)},
		{Name: "issues", URL: fmt.Sprintf("https://%s/issues/new/choose", repo)},
		{Name: "support", URL: fmt.Sprintf("https://%s/blob/%s/.github/SUPPORT.md", repo, branch)},
		{Name: "contributing", URL: fmt.Sprintf("https://%s/blob/%s/.github/CONTRIBUTING.md", repo, branch)},
		{Name: "security", URL: fmt.Sprintf("https://%s/security/policy", repo)},
	}...)

	return links
}

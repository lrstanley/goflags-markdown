// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package clix

import (
	"fmt"
	"strings"
)

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

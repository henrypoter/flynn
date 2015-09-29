package main

import (
	"bytes"

	c "github.com/flynn/flynn/Godeps/_workspace/src/github.com/flynn/go-check"
	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/pkg/cluster"
)

type TaffyDeploySuite struct {
	Helper
}

var _ = c.ConcurrentSuite(&TaffyDeploySuite{})

func (s *TaffyDeploySuite) deployWithTaffy(t *c.C, app *ct.App, github map[string]string) {
	client := s.controllerClient(t)

	taffyRelease, err := client.GetAppRelease("taffy")
	t.Assert(err, c.IsNil)

	rwc, err := client.RunJobAttached("taffy", &ct.NewJob{
		ReleaseID:  taffyRelease.ID,
		ReleaseEnv: true,
		Cmd: []string{
			app.Name,
			github["clone_url"],
			github["branch"],
			github["rev"],
		},
		Meta: map[string]string{
			"github":      "true",
			"github_user": github["user"],
			"github_repo": github["repo"],
			"branch":      github["branch"],
			"rev":         github["rev"],
			"clone_url":   github["clone_url"],
			"app":         app.ID,
		},
	})
	t.Assert(err, c.IsNil)
	attachClient := cluster.NewAttachClient(rwc)
	var outBuf bytes.Buffer
	exit, err := attachClient.Receive(&outBuf, &outBuf)
	t.Log(outBuf.String())
	t.Assert(exit, c.Equals, 0)
	t.Assert(err, c.IsNil)
}

// This test emulates deploys in the dashboard app
func (s *TaffyDeploySuite) TestDeploys(t *c.C) {
	assertMeta := func(m map[string]string, k string, checker c.Checker, args ...interface{}) {
		v, ok := m[k]
		t.Assert(ok, c.Equals, true)
		t.Assert(v, checker, args...)
	}

	client := s.controllerClient(t)

	github := map[string]string{
		"user":      "flynn-examples",
		"repo":      "go-flynn-example",
		"branch":    "master",
		"rev":       "a2ac6b059e1359d0e974636935fda8995de02b16",
		"clone_url": "https://github.com/flynn-examples/go-flynn-example.git",
	}

	// initial deploy

	app := &ct.App{}
	t.Assert(client.CreateApp(app), c.IsNil)
	debugf(t, "created app %s (%s)", app.Name, app.ID)

	release := &ct.Release{
		Meta: map[string]string{
			"github":      "true",
			"github_user": github["user"],
			"github_repo": github["repo"],
		},
	}
	t.Assert(client.CreateRelease(release), c.IsNil)
	t.Assert(client.SetAppRelease(app.ID, release.ID), c.IsNil)

	s.deployWithTaffy(t, app, github)

	release, err := client.GetAppRelease(app.ID)
	t.Assert(err, c.IsNil)
	t.Assert(release, c.NotNil)
	t.Assert(release.Meta, c.NotNil)
	assertMeta(release.Meta, "git", c.Equals, "true")
	assertMeta(release.Meta, "clone_url", c.Equals, github["clone_url"])
	assertMeta(release.Meta, "branch", c.Equals, github["branch"])
	assertMeta(release.Meta, "rev", c.Equals, github["rev"])
	assertMeta(release.Meta, "taffy_job", c.Not(c.Equals), "")
	assertMeta(release.Meta, "github", c.Equals, "true")
	assertMeta(release.Meta, "github_user", c.Equals, github["user"])
	assertMeta(release.Meta, "github_repo", c.Equals, github["repo"])

	// second deploy

	github["rev"] = "2bc7e016b1b4aae89396c898583763c5781e031a"

	release, err = client.GetAppRelease(app.ID)
	t.Assert(err, c.IsNil)

	release = &ct.Release{
		Env:       release.Env,
		Processes: release.Processes,
		Meta:      release.Meta,
	}
	t.Assert(client.CreateRelease(release), c.IsNil)
	t.Assert(client.SetAppRelease(app.ID, release.ID), c.IsNil)

	s.deployWithTaffy(t, app, github)

	newRelease, err := client.GetAppRelease(app.ID)
	t.Assert(err, c.IsNil)
	t.Assert(newRelease.ID, c.Not(c.Equals), release.ID)
	release.Env["SLUG_URL"] = newRelease.Env["SLUG_URL"] // SLUG_URL will be different
	t.Assert(release.Env, c.DeepEquals, newRelease.Env)
	t.Assert(release.Processes, c.DeepEquals, newRelease.Processes)
	t.Assert(newRelease, c.NotNil)
	t.Assert(newRelease.Meta, c.NotNil)
	assertMeta(newRelease.Meta, "git", c.Equals, "true")
	assertMeta(newRelease.Meta, "clone_url", c.Equals, github["clone_url"])
	assertMeta(newRelease.Meta, "branch", c.Equals, github["branch"])
	assertMeta(newRelease.Meta, "rev", c.Equals, github["rev"])
	assertMeta(newRelease.Meta, "taffy_job", c.Not(c.Equals), "")
	assertMeta(newRelease.Meta, "github", c.Equals, "true")
	assertMeta(newRelease.Meta, "github_user", c.Equals, github["user"])
	assertMeta(newRelease.Meta, "github_repo", c.Equals, github["repo"])
}

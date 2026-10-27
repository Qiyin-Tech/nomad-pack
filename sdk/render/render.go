// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

// Package render loads and renders local Packs without invoking the CLI.
package render

import (
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/nomad-pack/internal/pkg/manager"
	"github.com/hashicorp/nomad-pack/sdk/pack"
	"github.com/hashicorp/nomad/api"
)

// Options explicitly selects every input. No environment overrides are read
// implicitly. A nil Client disables template functions that query Nomad.
type Options struct {
	Path          string
	VariableFiles []string
	Variables     map[string]string
	Environment   map[string]string
	Client        *api.Client
	ParserV1      bool
	AllowUnset    bool
	Format        bool
	Auxiliary     bool
}

// Result retains warnings and template names, including dependent Packs.
// On failure it contains diagnostics but no partially usable templates.
type Result struct {
	Templates   map[string]string
	Metadata    *pack.Metadata
	Diagnostics hcl.Diagnostics
}

// Render processes a local Pack with the same loader, parser and renderer as
// nomad-pack. The caller owns source acquisition and output presentation.
func Render(opts Options) (*Result, error) {
	pm := manager.NewPackManager(&manager.Config{
		Path: opts.Path, VariableFiles: opts.VariableFiles,
		VariableCLIArgs: opts.Variables, VariableEnvVars: opts.Environment,
		UseParserV1: opts.ParserV1, AllowUnsetVars: opts.AllowUnset,
	}, opts.Client)
	rendered, failures := pm.ProcessTemplates(opts.Auxiliary, opts.Format, false)
	out := &Result{Metadata: pm.Metadata(), Diagnostics: pm.Diagnostics()}
	var errs []error
	for _, failure := range failures {
		errs = append(errs, failure.Err)
	}
	if err := errors.Join(errs...); err != nil {
		return out, err
	}
	out.Templates = rendered.ParentRenders()
	for name, content := range rendered.DependentRenders() {
		out.Templates[name] = content
	}
	return out, nil
}

package main

import (
	"go/ast"
	"go/token"

	"github.com/liut0/gomultilinter/api"
	"golang.org/x/net/context"
)

type vetLinterFactory struct{}

type vetLinterConfig struct {
	All      bool     `json:"all"`
	Enabled  []string `json:"enabled"`
	Disabled []string `json:"disabled"`
}

type vetLinter struct {
	chkr map[ast.Node][]func(*File, ast.Node)
}

var LinterFactory api.LinterFactory = &vetLinterFactory{}

func (f *vetLinterFactory) NewLinterConfig() api.LinterConfig {
	return &vetLinterConfig{
		All: true,
	}
}

func stringsToMap(vs []string) map[string]bool {
	m := make(map[string]bool, len(vs))
	for _, elem := range vs {
		m[elem] = true
	}
	return m
}

func (cfg *vetLinterConfig) NewLinter() (api.Linter, error) {
	l := &vetLinter{
		chkr: make(map[ast.Node][]func(*File, ast.Node)),
	}

	disabled := stringsToMap(cfg.Disabled)
	enabled := stringsToMap(cfg.Enabled)

	for typ, set := range checkers {
		for name, fn := range set {
			if disabled[name] {
				continue
			}
			if enabled[name] || (cfg.All && !experimental[name]) {
				l.chkr[typ] = append(l.chkr[typ], fn)
			}
		}
	}

	return l, nil
}

func (*vetLinter) Name() string {
	return "vet"
}

func (l *vetLinter) LintFile(ctx context.Context, file *api.File, reporter api.IssueReporter) error {
	f := &File{
		fset:    file.FSet,
		content: []byte{},
		name:    file.ASTFile.Name.Name,
		file:    file.ASTFile,
		WarnOverride: func(pos token.Pos, msg string) {
			reporter.Report(&api.Issue{
				Position: file.FSet.Position(pos),
				Severity: api.SeverityWarning,
				Message:  msg,
				Category: "vet",
			})
		},
	}

	pkg := new(Package)
	pkg.path = file.ASTFile.Name.Name
	pkg.files = []*File{f}
	pkg.typesPkg = file.PkgInfo.Pkg
	pkg.types = file.PkgInfo.Types

	f.pkg = pkg
	f.checkers = l.chkr
	f.walkFile(f.name, f.file)

	return nil
}

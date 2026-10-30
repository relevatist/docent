// Package assets holds files that docent installs or documents.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:claude-plugin
var claudePlugin embed.FS

// ClaudePlugin is the Claude Code plugin: the skill and the reload hook.
var ClaudePlugin, _ = fs.Sub(claudePlugin, "claude-plugin")

//go:embed claude-plugin/skills/docent/SKILL.md
var Skill string

//go:embed omp/docent-reload.ts
var OmpExtension string

//go:embed zellij/keys.kdl
var ZellijKeys string

//go:embed zellij/layout.kdl
var ZellijLayout string

// ZellijPlugin is the docent Zellij plugin. Build it with `make plugin`.
//
//go:embed zellij/docent.wasm
var ZellijPlugin []byte

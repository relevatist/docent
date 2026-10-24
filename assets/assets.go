// Package assets holds files that zide installs or documents.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:claude-plugin
var claudePlugin embed.FS

// ClaudePlugin is the Claude Code plugin: the skill and the reload hook.
var ClaudePlugin, _ = fs.Sub(claudePlugin, "claude-plugin")

//go:embed claude-plugin/skills/zide/SKILL.md
var Skill string

//go:embed omp/zide-reload.ts
var OmpExtension string

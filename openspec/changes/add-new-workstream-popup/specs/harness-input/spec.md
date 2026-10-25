# Spec Delta

## MODIFIED Requirements

### Requirement: Send a file/line reference
`zide send ref <file>[:<start>[-<end>]]` SHALL paste a reference to that file and line range into the harness input in the caller's tab. The reference format SHALL match the harness kind: `@file:start-end` for `claude` and `omp`, and `file:start-end` for other harnesses. The kind SHALL be the kind shown in the harness pane command, or else the base name of the first word of that command. A single-line range SHALL be written as `file:start`. The reference SHALL end with one space.

#### Scenario: Range reference to Claude Code
- **WHEN** the harness is `claude` and the user runs `zide send ref src/a.go:10-20`
- **THEN** the harness input receives `@src/a.go:10-20 `

#### Scenario: Single line
- **WHEN** the harness is `claude` and the user runs `zide send ref src/a.go:10-10`
- **THEN** the harness input receives `@src/a.go:10 `

#### Scenario: Unknown harness
- **WHEN** the harness is `aider` and the user runs `zide send ref src/a.go:10-20`
- **THEN** the harness input receives `src/a.go:10-20 `

#### Scenario: Wrapped harness with a kind
- **WHEN** the tab was launched with an alias whose command is `sandbox-exec -f omp.sb omp` and whose kind is `omp`, and the user runs `zide send ref src/a.go:10-20`
- **THEN** the harness input receives `@src/a.go:10-20 `

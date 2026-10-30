# Spec Delta

## ADDED Requirements

### Requirement: Pane writes keep the scroll position
When the docent Zellij plugin runs in the session, a docent command that writes to a pane (keys, commands, or pasted text) SHALL NOT change the scroll position of any pane other than the target pane. Without the plugin, docent SHALL still write to the target pane, and the scroll position of the focused pane MAY change.

#### Scenario: Reload from a background tab
- **WHEN** the user scrolls up in the harness pane of the visible tab, and the harness in a background tab edits a file and runs `docent reload`
- **THEN** the editor in the background tab reloads, and the visible harness pane stays at the same scroll position

#### Scenario: Open from the same tab
- **WHEN** the user scrolls up in the shell pane, and the harness in the same tab runs `docent open a.go`
- **THEN** the editor shows `a.go`, and the shell pane stays at the same scroll position

#### Scenario: Two clients attached
- **WHEN** two terminals are attached to the session and the harness runs `docent open a.go:3`
- **THEN** the editor receives the command once

#### Scenario: Plugin not loaded
- **WHEN** the session was started before `docent install zellij`, and the harness runs `docent reload`
- **THEN** the editor reloads and docent exits zero

### Requirement: Pane writes are unchanged
Through the plugin or without it, the target pane SHALL receive the same input as before: keys as the Zellij `send-keys` action encodes them, and pasted text inside one bracketed paste.

#### Scenario: Editor in insert mode
- **WHEN** the editor is in insert mode and `docent open a.go:3` runs through the plugin
- **THEN** no text is inserted into the current buffer and the editor shows `a.go` at line 3

#### Scenario: Send text through the plugin
- **WHEN** `docent send text` receives `a` ESC `[201~` CR `b`
- **THEN** the harness input receives `a[201~` CR `b` inside one paste and the harness does not submit it

# Tasks

## 1. Tour model without diffs

- [x] 1.1 In `internal/tour/tour.go`, remove `Base`, `Line`, the line kinds, `NotCovered`, `Item.Title`, `Item.Body`, `covers`, and `DiffTour`. Change `Build(t, cwd)` to drop the hunks parameter. Make `file` optional: a step needs `file` or `note`, and `lines` needs `file`. Verify with `tour_test.go` cases for a note-only step, an empty step, lines without file, a tour with `base` (ignored), and the invalid-step cases that exist now.
- [x] 1.2 In `internal/tour/model.go`, make sure `o` and `s` do nothing and the editor does not move on a step without a file. Verify with a `model_test.go` case.
- [x] 1.3 Delete `internal/tour/diff.go`, its tests, and `internal/tour/testdata/*.diff`. Verify `go build ./...` and `go vet ./...` pass.

## 2. Commands

- [x] 2.1 Delete `cmd/zide/diff.go` and its tests. Remove `diff` from the command table in `cmd/zide/main.go`, and remove `--diff` from `loadTourView`. Verify `zide --help` does not list `diff` (`main_test.go`).
- [x] 2.2 In `cmd/zide/tour.go`, remove the `tour.Changes` check from `startTour`, and update `tourUsage` (no `base`, optional `file`, the new keys). Verify `tour_test.go` passes outside a git repository.

## 3. Markdown view

- [x] 3.1 Add `charm.land/glamour/v2` v2.0.1 and `github.com/AlexanderGrooff/mermaid-ascii` (latest pseudo-version) to `go.mod`. Verify `go mod tidy` leaves no gin or cobra in the `go list -deps ./cmd/zide` output.
- [x] 3.2 In `internal/tui`, add `renderNote(note, width)`: `safe`, then mermaid fences to text blocks (with recover, and the source plus the reason on failure), then glamour with word wrap. Discard logrus output. Verify with tests: a `graph LR; A --> B` fence draws boxes, a bad fence keeps its source and shows the reason, an escape sequence becomes U+FFFD, and the output has no escape codes from mermaid-ascii.
- [x] 3.3 Change the step view to show the header and then the cached rendered note. Render again only on a step or width change. Use the tour title as the heading for a note-only step. Choose the dark or light glamour style from `tea.BackgroundColorMsg`. Verify with `tui_test.go` that a file step shows no file lines and that a note-only step shows the title.

## 4. Scroll keys

- [x] 4.1 Add `pgdown`/`space`/`ctrl+d`, `pgup`/`b`/`ctrl+u`, and `g`/`G`. Clamp scroll to `[0, len(lines)-room]`, and reset it on a step change. Update the help line. Verify with `tui_test.go` cases from the spec: page down on a 50-line note, `G`, `pgup` at the top, and reset after `n`.

## 5. Skill and docs

- [x] 5.1 Rewrite the tour section of `assets/claude-plugin/skills/zide/SKILL.md`: the note is Markdown, `file` is optional (overview steps), and there is no `base`. Add the show-me views with short examples (pseudocode, call tree, component tree, file tree, `diff` sketches, code block, and Mermaid flowchart, sequence, and ER) and guidance to keep diagrams narrow. Do not mention HTML. Remove the `zide diff` section. Verify `assets_test.go` passes and `grep -i html` on the skill finds nothing.
- [x] 5.2 Update `README.md`: remove `zide diff` and `base`, describe Markdown notes, Mermaid, and the new keys. Verify with `grep -n "zide diff\|base" README.md`.

## 6. Integration

- [x] 6.1 Run `make check`. Verify it passes, including the integration test where `zellij` and `hx` are on PATH.
- [x] 6.2 Run a tour with an overview step, a `diff` fence, and a `mermaid` fence in a real workstream. Verify that the rendering, paging, and editor sync work, and that the editor does not move on the overview step.

## Workflow follow-up

- Before archive, edit the `change-tour` Purpose in `openspec/specs/change-tour/spec.md` to remove "no change left out".
- Archive the change after review.

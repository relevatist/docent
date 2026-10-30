# Spec Delta

## ADDED Requirements

### Requirement: Layout of the new tab
`docent new` SHALL open the new tab with the layout of the current tab: the file in the `DOCENT_LAYOUT` marker of its harness pane, or the built-in layout when the harness pane has no marker. When the current tab has no harness pane, docent SHALL use the config `layout`, else the built-in layout. The picker SHALL NOT show a layout choice.

#### Scenario: Tab with a custom layout
- **WHEN** the current tab was opened with `docent --layout ~/layouts/pair.kdl`, and the user runs `docent new` and confirms a directory
- **THEN** the new tab opens with the layout in `~/layouts/pair.kdl`

#### Scenario: Tab with the built-in layout
- **WHEN** the current tab was opened with the built-in layout, the config now has `layout = "wide.kdl"`, and the user runs `docent new` and confirms a directory
- **THEN** the new tab opens with the built-in layout

#### Scenario: Not a docent tab
- **WHEN** the current tab has no harness pane, the config has `layout = "wide.kdl"`, and the user runs `docent new` and confirms a directory
- **THEN** the new tab opens with the layout in `wide.kdl` in the docent config directory

#### Scenario: Layout file deleted
- **WHEN** the current tab was opened with `docent --layout ~/layouts/pair.kdl`, that file has since been deleted, and the user runs `docent new` and confirms a directory
- **THEN** docent prints an error that names `~/layouts/pair.kdl` and opens nothing

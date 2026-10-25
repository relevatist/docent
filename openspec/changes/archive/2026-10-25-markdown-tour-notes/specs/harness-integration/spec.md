# Spec Delta

## ADDED Requirements

### Requirement: Skill explains how to write step notes
The skill SHALL tell the harness to write each step note in Markdown that explains the change at that step. The skill SHALL include the views the harness can use, each with an example: pseudocode, call trees, component trees, file trees, `diff` sketches of these, code blocks, and Mermaid flowchart, sequence, and ER diagrams. The skill SHALL be complete without other skills installed. The skill SHALL NOT suggest HTML or other output that the tour pane cannot show.

#### Scenario: Harness writes a tour
- **WHEN** the harness loads only the zide skill and writes a tour
- **THEN** the skill gives it the note format and the view examples

#### Scenario: No HTML
- **WHEN** the harness reads the skill
- **THEN** the skill does not suggest HTML files or opening a browser

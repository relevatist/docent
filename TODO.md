# TODO

Goal: help developers understand the code that they generate and review.

## Intent

- [ ] Decision log: the choices the agent made and the options it rejected, shown in the tour.
- [ ] Assumptions to confirm: the agent lists its assumptions, and the user accepts or rejects each one.
- [ ] Request trace: each step points to the part of the request that it does. Show parts of the request that no step covers.
- [ ] Scope check: flag changes outside the task the user asked for.

## Comprehension checks

- [ ] Quiz: "what happens when X?" questions after a tour.
- [ ] Explain it back: the user summarises the change, and the agent finds the gaps.

## Attention

- [ ] Risk tags: the agent tags steps (removed check, concurrency, security, migration). Sort or filter the tour by risk.
- [ ] Impact list: code that depends on the changed code but did not change.
- [ ] Coverage check: parts of the diff that no tour step covers.
- [ ] Familiarity tracking: remember what the user toured or opened, so tours spend more time on areas that are new to the user.
- [ ] Hotspots: files with high churn or many authors in git history get more attention.
- [ ] Test gaps: changed code that no test touches.
- [ ] Size budget: warn when a step or tour is too large to understand in one pass, and ask the agent to split it.
- [ ] New dependencies: flag new imports and libraries, with a one-line summary of each.

## Mental models

- [ ] Executable steps: a step runs a command (test, example, curl) in the shell pane.
- [ ] Before and after diagrams: package graph or call tree from static analysis, for the base and for HEAD.
- [ ] Repo map: entry points, main types and data flow for an unfamiliar codebase.
- [ ] Glossary: domain terms in the repo, each mapped to its type or function.
- [ ] Trace tour: run a test with tracing, and turn the call path into tour steps.
- [ ] Symbol peek: show the definition or callers of a symbol in the tour pane.
- [ ] State diagrams: states and transitions for code with state machines or lifecycles.

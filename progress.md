# Progress

## Working Rules

- Follow `plan.md` as the active execution baseline.
- Use `enhancement.md` as the primary design reference and use targeted reads from `gocode/` and `microsoft/vscode-copilot-chat` only to confirm concrete implementation decisions.
- Do not add tests.
- Do not weaken cwd containment, deterministic exact-edit behavior, or file-history rollback support for the sake of parity.
- Do not plan teams, swarm orchestration, remote execution, browser automation, or unrelated product lines.
- After each completed task: update this file, run formatting, and create a git commit.

## Current Status

| Workstream                               | Status      | Scope | Notes                                                                                                                         |
| ---------------------------------------- | ----------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| Enhancement planning baseline            | completed   | S     | 2026-04-12 created a new execution baseline from `enhancement.md` focused on file-tool robustness and subagent orchestration. |
| Phase 1 file semantics and safety        | not started | L     | Create vs overwrite semantics, read hardening, risk-tier approval, and write-surface consistency are planned but not started. |
| Phase 2 edit engine hardening            | not started | L     | `apply_patch`, edit failure taxonomy, recovery hints, and post-edit diagnostics are planned but not started.                  |
| Phase 3 subagent lineage and metadata    | not started | M     | Stable invocation ids, structured child metadata, and stronger TUI attribution are planned but not started.                   |
| Phase 4 child lifecycle and policy hooks | not started | M     | Shared loop alignment plus subagent start/stop hooks and block-stop reasons are planned but not started.                      |

## Completion Dashboard

This section is the canonical phase tracker. A phase is only complete when its `Remaining to Finish` list is empty and its exit criteria are all satisfied.

### Phase 1: File Semantics and Safety Hardening

**Status:** not started

**Landed**

- None yet.

**Remaining to Finish**

- Separate create, overwrite, and edit intent in the file-tool surface.
- Improve `file_read` partial-read guidance and binary-aware handling.
- Add risk-tiered approval rules for sensitive local paths without weakening cwd containment.
- Ensure stable diff-preview behavior and file-history coverage across all direct write paths.

**Exit Criteria Check**

- [ ] Create, overwrite, and edit intent are no longer conflated.
- [ ] `file_read` safely distinguishes text reads from binary-like content and guides continued reads.
- [ ] Higher-risk local files require stronger approval than ordinary source files.
- [ ] File-history behavior still covers every direct write path.

### Phase 2: Edit Engine Hardening

**Status:** not started

**Landed**

- None yet.

**Remaining to Finish**

- Add `apply_patch` as a dedicated patch-grade edit path.
- Standardize edit failure taxonomy and recovery hints across the edit family.
- Define explicit tool-selection guidance for exact edit, multi-replace, and patch flows.
- Surface post-edit diagnostics when available.

**Exit Criteria Check**

- [ ] `gocode` has a dedicated patch-grade edit path for larger structural changes.
- [ ] File-edit failures are machine-distinguishable and actionable.
- [ ] The edit ladder is explicit enough that structural edits do not overuse exact replacement tools.
- [ ] Post-edit diagnostics catch broken edits earlier when the environment can provide them.

### Phase 3: Subagent Lineage and Metadata

**Status:** not started

**Landed**

- None yet.

**Remaining to Finish**

- Add stable invocation ids across child launch, status, transcript, IPC, and cost surfaces.
- Extend `agent` and `agent_status` with structured child metadata.
- Update the TUI to render structured child state directly.

**Exit Criteria Check**

- [ ] Every child run is traceable from parent launch through completion or cancellation.
- [ ] Background and sync child runs expose enough metadata to debug them without opening raw transcripts by default.
- [ ] Child-agent lifecycle and cost data remain attributable without ambiguity.

### Phase 4: Shared Child Lifecycle and Policy Hooks

**Status:** not started

**Landed**

- None yet.

**Remaining to Finish**

- Align child execution more explicitly with the main agent-loop contracts.
- Add optional subagent start hooks.
- Add optional subagent stop hooks with block-stop reasons.
- Surface block-stop reasons in child status and transcript state.

**Exit Criteria Check**

- [ ] Parent and child execution share the same core lifecycle concepts instead of drifting.
- [ ] Child agents can be prevented from stopping early through explicit hook policy.
- [ ] Child stop-block reasons remain visible and actionable.
- [ ] Child isolation remains clear enough that parent context and artifact ownership are not polluted.

## Task Log

### 2026-04-12

- Completed: replaced the deleted planning files with a new enhancement execution baseline derived from `enhancement.md`.
- Completed: narrowed the roadmap to two workstreams only: file-tool robustness and subagent orchestration.
- Completed: translated the enhancement research into four implementation phases with explicit guardrails and exit criteria.
- Note: this remains a planning-only task. No runtime, tool, or TUI implementation work has started yet.

## Next Planning Baseline

See `enhancement.md` for the underlying research and decision rationale. See `plan.md` for the active execution sequence and phase scope.

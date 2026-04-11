# Enhancement Execution Plan

## Goal

Turn the research in `enhancement.md` into an implementation roadmap focused on two linked outcomes:

- make `gocode` file-related tools at least as robust as the strongest parts of VS Code Copilot Chat, while preserving `gocode`'s stricter local safety model where it is already better
- tighten subagent orchestration so child runs have clearer lineage, status, and lifecycle control without adding team, swarm, or remote complexity

## Primary Baseline

This plan is derived from `enhancement.md` and should be read together with it.

The key decision from the research is the execution bar for file tools:

- match or exceed VS Code Copilot Chat where it is stronger
- preserve `gocode` behavior where it is already safer or more deterministic
- do not copy flexible editor behaviors that weaken local CLI determinism

## Reference Basis

Primary planning reference:

- `enhancement.md`

Current local seams most relevant to this roadmap:

- `gocode/internal/tools/file_read.go`
- `gocode/internal/tools/file_write.go`
- `gocode/internal/tools/file_edit.go`
- `gocode/internal/tools/multi_replace_file_content.go`
- `gocode/internal/tools/file_diff_preview.go`
- `gocode/internal/tools/file_history.go`
- `gocode/internal/tools/file_history_tools.go`
- `gocode/internal/tools/path_resolution.go`
- `gocode/internal/tools/validation.go`
- `gocode/internal/tools/agent.go`
- `gocode/internal/agent/loop.go`
- `gocode/internal/agent/query_stream.go`
- `gocode/internal/ipc/`
- `gocode/tui/src/`

Targeted VS Code Copilot Chat reference anchors:

- `src/extension/tools/node/applyPatchTool.tsx`
- `src/extension/tools/node/applyPatch/parser.ts`
- `src/extension/tools/node/readFileTool.tsx`
- `src/extension/tools/node/createFileTool.tsx`
- `src/extension/tools/node/replaceStringTool.tsx`
- `src/extension/tools/node/multiReplaceStringTool.tsx`
- `src/extension/tools/node/editFileToolUtils.tsx`
- `src/extension/tools/node/searchSubagentTool.ts`
- `src/extension/tools/node/executionSubagentTool.ts`
- `src/extension/prompt/node/searchSubagentToolCallingLoop.ts`
- `src/extension/prompt/node/executionSubagentToolCallingLoop.ts`
- `src/extension/intents/node/toolCallingLoop.ts`

## Non-Negotiable Guardrails

- Keep strict working-directory containment as the default local safety model.
- Preserve deterministic exact-edit behavior as the default. Do not silently introduce fuzzy or similarity-based edits.
- Keep file-history snapshot and rewind support compatible with every future direct-write tool.
- Keep create, overwrite, and edit semantics explicit enough that permission and failure handling stay understandable.
- Keep child agents bounded, local-first, and artifact-safe.
- Do not add team agents, swarm behavior, remote execution, browser automation, or unrelated product lines.
- This plan authorizes sequencing and scope only. It does not authorize weakening permissions, budgeting, or transcript clarity for convenience.

## File-Tool Success Bar

The file-tool work in this plan is only complete when all of the following are true:

1. Exact edits remain deterministic and reject ambiguous matches.
2. Larger structural edits have a dedicated patch-grade path.
3. Create, overwrite, and edit intent are clearly separated.
4. Every direct write path emits a stable diff preview.
5. Edit failures are classified and return actionable recovery hints.
6. Reads detect binary or image-like inputs and fail safely or switch modes.
7. Large or partial reads tell the model how to continue.
8. Risky paths inside the allowed tree require stronger confirmation than ordinary source files.
9. File-history snapshot and rewind behavior still works across all write tools.
10. Post-edit diagnostics are surfaced when available.

## Phase 1: File Semantics and Safety Hardening

**Purpose:** close the semantic and safety gaps in the current file-tool surface before adding a more flexible edit engine.

### Scope

- Separate file creation from overwrite and from in-place editing.
- Decide whether `file_write` becomes explicit about overwrite policy or whether a distinct create tool is introduced.
- Remove or de-emphasize implicit file creation through `file_edit` when `old_string` is empty.
- Improve `file_read` so large and partial reads include continuation guidance.
- Add binary or image-like file detection in `file_read` with a safe fallback behavior.
- Add risk-tiered confirmation rules inside the allowed tree for dotfiles, editor config, shell config, and similarly sensitive paths.
- Keep current cwd containment intact while adding the finer-grained risk tiers.
- Make diff-preview behavior consistent across all direct write tools that modify file content.
- Confirm that file-history tracking still wraps every direct write path.

### Exit Criteria

- Create, overwrite, and edit intent are no longer conflated.
- `file_read` safely distinguishes text reads from binary-like content and tells the model how to continue partial reads.
- Higher-risk local files require stronger approval than ordinary source files.
- The existing `file_history` and `file_history_rewind` workflow still covers every direct write path.

## Phase 2: Edit Engine Hardening

**Purpose:** add a stronger edit ladder without weakening deterministic exact-edit behavior.

### Scope

- Add an `apply_patch` tool for multi-hunk and multi-file structural edits.
- Define tool-selection guidance for when to use:
  - `file_edit`
  - `multi_replace_file_content`
  - `apply_patch`
- Standardize edit failure kinds across the edit family:
  - no match
  - multiple matches
  - no-op edit
  - invalid edit format or unsupported operation
- Return recovery hints with edit failures so the model can reread, narrow context, or switch tools intentionally.
- Add optional post-edit diagnostics to file-modifying tools when the environment can provide them.
- Keep exact replacement strict by default. Any edit healing or fuzzy behavior, if explored later, must never become the silent default path.

### Exit Criteria

- `gocode` has a dedicated patch-grade edit path for larger structural changes.
- File-edit failures are machine-distinguishable and actionable.
- The model has a clear edit ladder instead of overusing exact replacement for structural changes.
- Post-edit diagnostics are visible enough to catch broken edits earlier.

## Phase 3: Subagent Lineage and Metadata

**Purpose:** make child-agent execution easier to follow, inspect, and attribute.

### Scope

- Assign one stable invocation id per child-agent run.
- Propagate that id through:
  - parent tool results
  - child transcript metadata
  - status polling
  - IPC events
  - TUI state
  - cost summaries
- Extend `agent` and `agent_status` results with structured metadata beyond final text, such as:
  - phase or lifecycle state
  - active or last tool
  - transcript path
  - result path
  - last meaningful status message
- Make the TUI consume structured child metadata directly instead of inferring state from loosely structured summaries.

### Exit Criteria

- Every child run can be traced cleanly from parent launch through completion or cancellation.
- Background and synchronous child runs expose enough structured metadata to debug them without opening raw transcript files by default.
- Child-agent cost and lifecycle state are attributable without ambiguity.

## Phase 4: Shared Child Lifecycle and Policy Hooks

**Purpose:** make child-agent behavior align with the main loop while preserving isolation.

### Scope

- Align child execution more explicitly with the main `internal/agent` loop contracts.
- Keep child history, token budgeting, and permission behavior isolated from the parent session.
- Add optional child start hooks that can inject additional child context.
- Add optional child stop hooks that can block completion with explicit reasons.
- Surface block-stop reasons back into child status and transcript state so they remain actionable rather than disappearing.
- Keep child tool allowlists explicit and role-specific.
- Preserve current local-first boundaries and artifact-safe result delivery.

### Exit Criteria

- Parent and child execution share the same core lifecycle concepts instead of drifting into separate runtimes.
- Child agents can be prevented from stopping early through explicit policy hooks.
- Child stop-block reasons remain visible in status and transcript state.
- Child isolation remains clear enough that parent context budgets and artifact ownership do not get polluted.

## Phase Ordering Rationale

- File-tool work comes first because it is the clearest robustness gap against the current research baseline.
- Semantic and safety hardening comes before patch editing so the more powerful edit path lands on top of clearer permissions, clearer failure behavior, and stronger read semantics.
- Subagent lineage comes before hook-based lifecycle work because visibility and attribution problems are easier to solve before policy surfaces are layered on top.

## Cross-Cutting Concerns

- **Artifact safety:** child agents must continue to return reports without directly mutating parent-owned artifacts by default.
- **Cost tracking:** child-agent lineage and post-edit diagnostics both add visibility requirements to the current cost and telemetry summaries.
- **TUI integration:** structured tool and child-agent outputs should be preferred over raw textual inference whenever state needs to be rendered.
- **Documentation:** README and prompt/tool descriptions must stay synchronized with any semantics change in file creation, overwrite rules, edit tool choice, or subagent lifecycle behavior.

## Out of Scope

- team or swarm orchestration
- remote agents or remote execution
- browser automation
- notebook-specific file editing work
- generic tool-count parity efforts that do not improve robustness

## Completion Standard

This roadmap is complete only when:

- the file-tool surface meets the match-or-exceed bar defined above
- the subagent lifecycle is traceable and policy-aware without adding product sprawl
- `gocode` keeps the local-first strengths that already exceed VS Code Copilot Chat in strict path safety, rollback support, and deterministic chunk validation

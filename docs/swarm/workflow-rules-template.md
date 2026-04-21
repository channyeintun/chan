# Workflow Rules Template

- At startup, confirm your assigned role, branch, and worktree.
- Work only in your assigned branch or workspace.
- Process one task at a time. Queue new work instead of silently dropping it.
- The architect clarifies scope, intent, and acceptance boundaries.
- The coder implements the agreed slice and prepares a clean handoff.
- The reviewer verifies behavior, risk, and clarity before acceptance.
- Pull or merge accepted changes only from the correct sending role.
- Every handoff must include:
  - a short summary of the change
  - the files or areas that matter
  - what was verified
  - risks or open questions
  - the next action for the receiving role
- Use Nami's swarm tools to move work forward:
  - `swarm_submit_handoff`
  - `swarm_list_inbox`
  - `swarm_update_handoff`

## Queueing Rule

- If new work arrives while you are already busy, finish the current slice first unless the user explicitly reprioritizes.
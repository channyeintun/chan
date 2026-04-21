# Nami Swarm Prompt Templates

These files are starting points for `.nami/swarm/` in a Nami project.

They are adapted from the layered constitution style and role discipline used in SwarmForge, reshaped for Nami's role overlays and swarm handoff tools.

## Copy Map

- `constitution-template.md` -> `.nami/swarm/constitution.md`
- `project-rules-template.md` -> `.nami/swarm/constitution/project.md`
- `engineering-rules-template.md` -> `.nami/swarm/constitution/engineering.md`
- `workflow-rules-template.md` -> `.nami/swarm/constitution/workflow.md`
- `architect-template.md` -> `.nami/swarm/roles/architect.md`
- `coder-template.md` -> `.nami/swarm/roles/coder.md`
- `reviewer-template.md` -> `.nami/swarm/roles/reviewer.md`

## How To Use

1. Copy the files you want into `.nami/swarm/`.
2. Replace bracketed placeholders with project-specific rules.
3. Keep the constitution files stable and shared across roles.
4. Keep each role file focused on ownership, startup behavior, handoffs, and what that role must avoid.

## Handoff Checklist

Use this shape whenever work moves from one role to another:

- what changed
- which files or areas matter
- what was verified
- risks or open questions
- the next action for the receiving role
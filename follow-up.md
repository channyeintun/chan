# Follow-Up Work

## Queue Policy Enforcement

The swarm spec model defines three queue policies per role (`fifo`, `batch-review`, `latest-wins`) but the runtime does not enforce them yet. `ListHandoffs` always returns all matching handoffs sorted by recency regardless of the role's declared policy.

### What needs to happen

1. Add a `DequeueHandoffs(store, sessionID, role, policy)` function in `internal/swarm/handoff.go` that applies the role's queue policy when a child agent reads its inbox:
   - `fifo`: return oldest pending first, one at a time.
   - `batch-review`: return all pending handoffs for the role in a single batch.
   - `latest-wins`: return only the most recent pending handoff, mark older pending items as superseded.
2. Wire `DequeueHandoffs` into the child agent inbox consumption path so `swarm_list_inbox` can optionally respect the policy.
3. Add a `superseded` status to the handoff status machine for `latest-wins` dropped items.

### Where the scaffolding already exists

- `QueuePolicy` type and normalization in `internal/swarm/spec.go`
- `ResolvedRole.QueuePolicy` is populated during spec resolution
- `ListHandoffs` and `UpsertHandoff` in `internal/swarm/handoff.go` handle the storage layer
- `swarm_list_inbox` tool in `internal/tools/swarm_handoffs.go` is the consumer entry point

# Read Tool Implementation Progress

## References

- Reviewed chan's current `read_file` implementation.
- Reviewed reference implementations from opencode, Claude Code, and VS Code Copilot Chat during planning.

## Task Status

1. Create progress tracker
   - Status: completed
   - Notes: Added this file to track task-by-task execution and commits.

2. Refactor `read_file` API
   - Status: pending
   - Notes: Move to `filePath` + `offset` + `limit`, add bounded default reads, byte cap, long-line clipping, and canonical continuation hints.

3. Add reread dedup state
   - Status: pending
   - Notes: Add session-scoped unchanged-slice suppression keyed by path, range, size, and modification time.

4. Invalidate cache on writes
   - Status: pending
   - Notes: Invalidate read-state entries after successful file mutations.

5. Tighten prompt guidance
   - Status: pending
   - Notes: Update tool description and engine system prompt guidance for canonical read behavior.

6. Format and verify changes
   - Status: pending
   - Notes: Run formatting after each completed task and check for relevant errors.

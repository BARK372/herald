# Workflow

Herald follows a simple async pattern: **start a task, check on it, get the result**. This page walks through real examples.

## The Core Loop

```
You (Claude Chat)          Herald                     Claude Code
─────────────────          ──────                     ───────────
"Refactor auth..."    ──►  start_task
                           → creates branch
                           → spawns Claude Code  ──►  reads codebase
                                                      refactors code
                                                      runs tests
                                                      commits changes
                      ◄──  task_id: herald-a1b2c3d4

"How's it going?"     ──►  check_task
                      ◄──  ✅ Completed (4m 12s)
                           4 files changed (+127/-23)

"Show me the diff"    ──►  get_diff
                      ◄──  auth/middleware.go
                           +func ValidateJWT(...)
                           -func CheckSession(...)
```

Three tools. That's the core loop. Start, check, get results — all from wherever you are.

## Example: Refactor a Module

### 1. Start the task

> *"Refactor the auth middleware in my-api to use JWT instead of session cookies. Run the tests when done."*

Herald responds immediately:

```
🚀 Task started

• ID: herald-a1b2c3d4
• Project: my-api
• Priority: normal
• Branch: herald/a1b2c3d4-refactor-auth

💡 Use check_task to monitor progress.
```

### 2. Check progress

A few minutes later, ask:

> *"How's the auth refactor going?"*

```
🔄 Running (2m 34s)

• Progress: Refactoring auth/middleware.go...
• Cost: $0.18 so far
```

### 3. Get the result

Once complete:

> *"Show me the result"*

```
✅ Task completed

• ID: herald-a1b2c3d4
• Duration: 4m 12s
• Cost: $0.34
• Turns: 8

Summary: Refactored auth middleware from session cookies to JWT.
Modified 4 files, all tests passing.

💡 Use get_diff to see changes, or get_result with format='full' for complete output.
```

### 4. Review the diff

> *"Show me the diff"*

Herald returns the full Git diff of the task branch.

## Example: Fix a Bug

> *"There's a nil pointer panic in the user handler when email is empty. Fix it and add a test."*

This is a single start → check → result cycle. Herald creates a branch, Claude Code fixes the bug, writes a test, and commits.

## Example: Multi-Turn Session

Herald supports session resumption for iterative work:

### First task

> *"Add a /health endpoint to my-api"*

The result includes a `session_id`. Herald tells Claude Chat about it.

### Follow-up

> *"Actually, also add a /ready endpoint that checks the database connection. Continue the same session."*

Claude Chat passes the `session_id` to `start_task`. Claude Code picks up where it left off, with full context of the previous work.

## Task Priorities

You can request different priority levels:

> *"This is urgent — fix the production 500 error in the payment handler"*

Priority levels: `low`, `normal` (default), `high`, `urgent`.

Urgent tasks jump to the front of the queue and execute before lower-priority tasks.

## Dry Runs

> *"Plan how you'd add rate limiting to the API, but don't make any changes"*

With `dry_run: true`, Claude Code analyzes and plans but doesn't modify files. Useful for reviewing an approach before committing to it.

## Task Lifecycle

```
pending → queued → running → completed
                           → failed
                           → cancelled

linked (created via herald_push, can be resumed with start_task)
```

| Status | Meaning |
|---|---|
| `pending` | Task created, not yet started |
| `queued` | Waiting in the priority queue (concurrency limit reached) |
| `running` | Claude Code is executing |
| `completed` | Finished successfully |
| `failed` | Claude Code encountered an error |
| `cancelled` | Cancelled by user via `cancel_task` |
| `linked` | Session pushed from Claude Code via `herald_push` — ready for remote continuation |

## Reverse Flow: Claude Code → Herald

Herald's bridge is **bidirectional**. When working in your terminal with Claude Code, you can push the session to Herald for later continuation from Claude Chat (e.g., from your phone):

### Push from terminal

Claude Code calls `herald_push` with the session context:

```
You (terminal)             Claude Code                Herald
──────────────             ───────────                ──────
"Push to Herald"      ──►  herald_push
                             → session_id, summary,
                               files, branch       ──►  linked task created
```

### Continue from phone

Later, from Claude Chat:

> *"What sessions are waiting for me?"*

`list_tasks` shows linked sessions. Then:

> *"Resume that session"*

Claude Chat calls `start_task` with the `session_id` from the linked task, picking up where Claude Code left off.

## What's Next

- [Tools Reference](tools-reference.md) — Complete parameter details for all 10 tools
- [Multi-Project](multi-project.md) — Working with multiple codebases

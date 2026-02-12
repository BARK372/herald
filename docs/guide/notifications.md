# Notifications

Herald pushes task lifecycle updates directly to Claude Chat via **MCP server notifications**. This uses the SSE channel that MCP Streamable HTTP maintains between Claude Chat and Herald — no external services needed.

## How It Works

When you start a task from Claude Chat, Herald pushes updates as they happen:

- **task.started** — Task began execution
- **task.progress** — Significant progress (tool changes, sub-agent activity)
- **task.completed** — Task finished successfully
- **task.failed** — Task failed with an error
- **task.cancelled** — Task was cancelled

Progress notifications are debounced (default: 3 seconds) to avoid flooding. Terminal events (completed, failed, cancelled) are always sent immediately.

## No Configuration Needed

MCP notifications are always enabled. They use the existing MCP SSE connection — no external services, no API keys, no setup.

## Targeted Delivery

Herald captures the MCP session ID when `start_task` is called. Notifications for a task are sent to the specific Claude Chat session that started it. If that session is no longer connected, Herald falls back to broadcasting to all connected clients.

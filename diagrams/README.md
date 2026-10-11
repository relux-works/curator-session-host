# Session-host figures

These PlantUML sources illustrate the [current specification](../spec/session-host.md).
They are design views, not implementation or provider qualification evidence.
Current/SH1 ownership is labelled separately from gated SH2+ behavior; conceptual
state labels do not define unpublished wire enums. Notes mark what the spec leaves
unspecified. Relative links inside each figure lead back to its defining sections.

| Audit row / figure | Source | Defining spec sections |
|---|---|---|
| M-SH1 process/container ownership | [containers.puml](containers.puml) | [§1](../spec/session-host.md#1-what-the-host-is), [§3.1–3.3](../spec/session-host.md#31-service-account-state-and-capabilities) |
| M-SH1 logical components | [components.puml](components.puml) | [§3](../spec/session-host.md#3-phase-sh2-platform-canons), [§3.8](../spec/session-host.md#38-goals-across-accounts) |
| M-SH2 controller recovery and restart | [session-recovery.puml](session-recovery.puml) | [§3.2–3.5](../spec/session-host.md#32-holders-under-agent-accounts-and-the-process-tree) |
| M-SH3 session, execution, goal and wake states | [session-goal-wake-states.puml](session-goal-wake-states.puml) | [§3.5](../spec/session-host.md#35-park-restart-emergency-stop-board-completion), [§3.7–3.8](../spec/session-host.md#37-wake-and-liveness) |
| M-SH3 approval and receipt states | [approval-receipt-states.puml](approval-receipt-states.puml) | [§3.4](../spec/session-host.md#34-effects-fences-and-admission), [§7](../spec/session-host.md#7-open-questions) |
| M-SH4 storage ownership | [storage-layout.puml](storage-layout.puml) | [§3.1](../spec/session-host.md#31-service-account-state-and-capabilities), [§3.4–3.5](../spec/session-host.md#34-effects-fences-and-admission) |
| M-SH4 extraction gate deployment | [sh1-gate-deployment.puml](sh1-gate-deployment.puml) | [§2.4–2.6](../spec/session-host.md#24-the-gate-three-lanes), [gate guide](../docs/sh1-gate.md#lanes-and-ownership) |

The inventory and kinds come from [audit B4, M-SH1–M-SH4](https://github.com/relux-works/curator/blob/48789217/.research/261010_platform-docs-audit.md#b4-missing-or-insufficient-module-diagrams--exact-proposed-files).
The current spec supersedes the audit's older projection-lane placement and
FD3/reconnect shorthand: lane 1 is public hosted CI with a private pinned board
checkout; FD3 carries the plan only, and the authenticated runner/host channel
remains an activation gate. No extra inherited descriptor is drawn.

Sources use basic PlantUML declarations, notes, links and control flow without
remote includes or themes. Rendering has not been performed on the development
host; no tests or builds run here. The configured Change Request validation runs
at role handoff.

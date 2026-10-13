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
| M-SH5 host wait/lock graph | [host-wait-lock.puml](host-wait-lock.puml) | [§3.4](../spec/session-host.md#34-effects-fences-and-admission), [§7](../spec/session-host.md#7-open-questions) |
| R-TRANSFER/R-VERIFY workspace round | [workspace-transfer.puml](workspace-transfer.puml) | [§3.1](../spec/session-host.md#31-service-account-state-and-capabilities), [§3.5](../spec/session-host.md#35-park-restart-emergency-stop-board-completion), [§7.13/17](../spec/session-host.md#7-open-questions) |
| R-ALLSTOP node-wide emergency stop | [node-emergency-stop.puml](node-emergency-stop.puml) | [§3.5](../spec/session-host.md#35-park-restart-emergency-stop-board-completion), [§7.16](../spec/session-host.md#7-open-questions) |
| M-SH4 extraction gate deployment | [sh1-gate-deployment.puml](sh1-gate-deployment.puml) | [§2.4–2.6](../spec/session-host.md#24-the-gate-three-lanes), [gate guide](../docs/sh1-gate.md#lanes-and-ownership) |

The inventory and kinds come from [audit B4, M-SH1–M-SH4](https://github.com/relux-works/curator/blob/48789217/.research/261010_platform-docs-audit.md#b4-missing-or-insufficient-module-diagrams--exact-proposed-files).
The current spec supersedes the audit's older projection-lane placement and
FD3/reconnect shorthand: lane 1 is public hosted CI with a private pinned board
checkout; FD3 carries the plan only, and the authenticated runner/host channel
remains an activation gate. No extra inherited descriptor is drawn.

The trusted standing-session lifetime decision of 2026-10-12, refined by R-STANDING on 2026-10-13, is reflected in M-SH1,
M-SH2 and the M-SH3 session/execution/wake view: no preset lifetime or runtime
limit; only explicit stop, operator action or own exit ends the session.
Park and controlled restart preserve it. The dispatcher's 12-hour maximum is
for non-standing executions only. Liveness deadlines and capability or approval
expiry do not impose a standing-session lifetime. R-MAINT in those views references
the dispatcher's bounded maintenance stop: the deadline bounds the attempt;
confirmed absence and reconciliation precede mode/state/binary changes. Unknown
or refused stop defers maintenance, allocations and identities stay, and R-MAINT
withdraws the permit; restart needs fresh owner admission. Recovery controls
remain reachable while forward admission is closed (F-15); no live supervisor
adoption or automatic expiry.

R-START/R-SESSION are reflected in M-SH1 ownership/components, M-SH2 recovery,
M-SH3 lifecycle and M-SH4 storage: durable host intent and closed fence before
`host_start`, a dispatcher-owned session-scoped **host admission permit** for one
`logical_session_id`, and the dispatcher as sole privileged start caller. Each
physical start keeps its own `(execution_id, controller_epoch, op_id)` claim,
fence and receipt interlock. Successors use the owner's **session lifecycle
authorisation**, confirmed prior-claim absence and fresh checks; replay of
`(logical_session_id, op_id)` returns the same successor. Controlled restart,
wedged park and emergency stop use `host_stop`, distinct from owner `cancel`
and granting no start authority. Fence changes withdraw or reconcile the affected
start's authority; host failure retains the claim and closed fence. Fresh runner
channel, current-epoch registration and readiness precede delivery.

The same views reference the user manager's §7 **fate table** (R-FATE), including
service-replacement affected sets and the fence against surviving dispatcher
writers, and F-13's single account-wide termination scope. Host deny-all egress
and its readiness prerequisites follow R-SVC-EGRESS/user manager §8; F-15 keeps
authorised controls reachable while admission is closed. M-SH5 publishes the
host's R-LOCK wait/lock graph: journal and record transitions hold the state lock;
synchronous dispatcher/runner calls do not. Per-record in-flight markers and
late-result revalidation preserve the fence. Publication and per-platform
qualification remain gates; none of these figures establishes survival or network
conformance. M-SH4 also references R-GEN restore fencing, R-REBIND credential
renewal, R-TIME and F-16/R-TRANSFER’s standalone cross-account repository with fresh local configuration and no hooks/remotes/credential helpers.

Wave 5 is reflected throughout the SH2+ views: R-STANDING restricts host sessions to trusted identities and checks plan surface digests before every launch/restart/native resume (or supplies root-owned surfaces); drift refuses and explicit re-materialization repairs. R-EXEC admits only the plan’s harness and qualified light tools. R-VERIFY moves builds/tests and worker-code execution to another machine; R-WATCH delivers a journaled stop with the reason recorded. R-ALLSTOP closes every delivery fence, snapshots bound boards and retains fences until confirmed absence, with explicit recovery only. The dedicated stop view shows R-OPERATOR custody and each service’s part; its arrows do not invent a coordination wire or crash ordering. Ownership/storage views reference the qualified R-UNTRUSTED, R-NET-TOOL, R-CEILINGS, R-RANGE-DENY, R-SOCKETS and R-SHARED prerequisites. The transfer view distinguishes admitted rework from an unchanged-plan successor. Formats/caps/codes remain open in §7.13–18. The SH1 gate deployment figure already places builds/tests in hosted CI, not an agent account, and is unchanged.

Sources use basic PlantUML declarations, notes, links and control flow without
remote includes or themes. Rendering has not been performed on the development
host; no tests or builds run here. The configured Change Request validation runs
at role handoff.

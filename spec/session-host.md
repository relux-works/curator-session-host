# swarma-session-host: specification

- **Status:** draft r9 (2026-10-12; accepted cycle-2 review findings). The session-manager package has not moved; hosted gate tooling is a separate prototype.
- **Revision:** r9 applies F-20/R-SESSION (session-scoped permit, `host_stop` and owner-authorised successors), F-15 (recovery controls while forward admission is closed), F-18/R-FATE, F-13 (single termination scope), F-24/R-SVC-EGRESS and F-28/R-LOCK, with F-14/R-REBIND, F-17/R-GEN and F-21/R-TIME cross-references. F-16 keeps cross-account repositories standalone; lifecycle, ownership and wait/lock figures follow the text.
- **Revision:** r8 applies accepted F-4/R-START (`host_start` backed by the host admission permit), F-2/R-MAINT (bounded dispatcher maintenance stop) and F-5 (dispatcher service transport from the first slice). R2 native-lock confirmation and agent-side custody remain; figures follow the revised start and maintenance ordering.
- **Revision:** r7 applies the owner decision of 2026-10-12: standing sessions and their executions have no preset lifetime or runtime limit; park and restart preserve the session. This supersedes the R3 runtime proposal and aligns the lifecycle figures.
- **Revision:** r6 assigns native-lock confirmation to the fresh runner (R2) and records unpublished goal surfaces under D-GOALS and the standing-session runtime question (R3).
- **Normative language:** MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.
- **Lineage:** continues the earlier session-host specification and the as-is extraction plan (revision 2, frozen as the reference alternative on 2026-10-05). The owner chose the as-is path on 2026-10-09: extract one to one first, then bring the host to the platform's rules. r3 closed the frozen plan's known gaps; r4 takes the architecture review of r3 (0 P0, 8 P1, 4 P2).
- **Companions:** [swarma-dispatcher](https://github.com/relux-works/swarma-dispatcher) (admission, allocations, executions, the session-host backend gate), [swarma-user-manager](https://github.com/relux-works/swarma-user-manager) (agent accounts, `swarma-user-launcher`, `exec.stop`), [swarma-credential-broker](https://github.com/relux-works/swarma-credential-broker), [curator-agent-launcher](https://github.com/relux-works/curator-agent-launcher) (`curator-run`), [agent-session-bridge](https://github.com/relux-works/agent-session-bridge), the platform architecture §5.1, §5.3, §7.5.
- **Contract authority:** the [platform contract table](https://github.com/relux-works/curator/blob/177cb554/.research/261010_platform-contract-table.md), read with the supplied owner decisions of 2026-10-10/11. D-GOALS settles present goal authorship; D-EXECUTOR settles the narrow one-shot executor and first broker consumer; D-PATH settles managed PATH; D-WIRE-NAMES settles wire naming; D-RESUME settles native-resume compatibility. These decisions supersede the corresponding open or older entries in that table. Unpublished schemas remain activation gates (§7).

## 1. What the host is

The session host keeps agent sessions alive in the background: it starts a harness in a given directory, lets a person attach and detach, delivers messages into a session, and reports events. A goal is optional. The module MUST work without goals, a board, an orchestrator or their toolchain. Task-board is an OPTIONAL client of a generic goal interface (§3.8). Today this is `tb-sessiond`, the session manager inside task-board (`tools/board-cli/internal/sessionmanager` in relux-works/skill-project-management).

| Phase | Service and logical ownership | Process ownership |
|---|---|---|
| Today / SH1 extraction | Board-composed `tb-sessiond`; the extracted module supplies its serving library through C1 | The daemon owns Claude's PTY master and a private Codex app-server per session |
| SH2+ target, after activation | `swarma-session-host` owns durable metadata, logs, receipts, fences and input admission; `swarma-dispatcher` owns admission, allocation, execution records and cleanup | One `swarma-session-runner` per session under `worker-<label>` owns its PTY master or private app-server transport and credential lease client |

Logical ownership by the host module does not mean that the service account owns the harness or its transport. The agent account generation, execution identity and host controller epoch are separate (§3.3).

Figures: [M-SH1 process/container ownership](../diagrams/containers.puml) and
[M-SH1 logical components](../diagrams/components.puml) distinguish current/SH1
from the gated SH2+ target. The component view groups specified responsibilities;
it does not define new internal APIs.

### 1.1 Today *(fact, at the pinned source)*

- One control plane per board, keyed by `BoardKey`; state in the user state directory; self-restoring start-up; one daemon per board.
- The daemon registers two provider factories: Claude (a PTY the daemon owns) and Codex (a private app-server per session). Codex goals are delivered as text directives; its native goals are disabled.
- The Muse code is present but not registered: the MSP provider host and the Muse PTY host (`musepty`) exist with their tests; their factory registration and command-line dispatch are later work. The hosted launch route decodes its input and always refuses capability admission.
- Goals are applied inside the board's write transaction; Claude acknowledges the exact rendered revision; deferred application has an acknowledgement waiter.
- Notices: a durable inbox per session, written directly by board producers, delivered only while a client is attached.
- A Claude session dies when the daemon is replaced (the daemon holds the PTY master).
- Access control: one bearer token in a private file; any process of the operator's user can use it.

## 2. Phase SH1: extraction as it is

The first step moves code, not behaviour.

### 2.1 What moves

The complete recursive snapshot of the package and its sub-packages at a pinned source commit, without the source repository's history (the commit is recorded with the move; the NOTICE keeps the original attribution):

| Group | Production files | Lines |
|---|---:|---:|
| Claude host, output, transcript, context | 9 | 7,852 |
| Codex host, context, goals, continuation | 4 | 4,295 |
| Muse MSP client, types, host, dispatch, schema, validation, text goals | 8 | 9,795 |
| hosted launch scaffold (closed route) | 6 | 964 |
| terminal (including resize) | 2 | 263 |
| notices | 2 | 513 |
| key, layout, records, log, liveness, resume, repair | 7 | 2,133 |
| host contract, types, launch plan | 3 | 1,151 |
| goal application, requester, activation, owner operation | 4 | 461 |
| client, control, daemon, manager, takeover, locks, shutdown | 13 | 6,083 |
| builder, context, materialisation, access, proxy, enforcement | 30 | 12,006 |
| `musepty` | 5 | 3,554 |
| **total production** | **93** | **49,070** |
| tests (root and `musepty`) | 175 | 95,031 |
| fixture programs and their test | 3 | 1,682 |

plus 18 non-Go files (the PTY grammar, the captured terminal corpus, schema and dispatch goldens, the hosted diagnostic registry). Inclusion is not activation: the Muse factories stay unregistered and the hosted route stays closed in SH1.

### 2.2 What stays in task-board

The board and its goal ledger, routing, the spawn runtime, run manifests, acceptance and handoff evidence, repository operations, the `task-board claude|codex|session|spawn` commands, and **the `tb-sessiond` executable and its composition root** (goal stores, run evidence, context authority, feature gates, local security policy) through C1. The module provides the serving library the composition root calls.

### 2.3 The seam

G, X and Y publish a source-derived disposition for **every** outward surface; an entry may be "retained compatibility service":

- **29/29** exported client methods, among them attach, stop, list, show, open terminal with initial geometry, resize terminal, owner activation, goal apply and rollback, run owner, the nine context-service methods, and `Close`, `Status`, `RequireGoalReflectionContract`, `Drain`, `Detach`, `Stop`, `TakeOverOwnerSlot`, `Doctor`, `ReconcileOwner`;
- **45/45** free functions the board's production files reference (connection and version, daemon and factory construction and replacement, launch-plan decoding and rebinding, context construction and gateway, notice, observation and operator-log helpers);
- **52/52** board and configuration library functions the host calls directly (222 call sites), each given a neutral local implementation or a narrowly typed injected service;
- the injected contracts: goal ledger (get, reconcile, confirm native applied) and binding (apply, rollback); `GoalPlaneGate`; the deferred goal binding with its acknowledgement waiter and cancellation; the context authority (11 methods and the read snapshot).

The file-backed notice API stays as it is in SH1. The goal contracts above describe the optional compatibility adapter: the host's generic goal interface must allow launch and session operation without a goal ledger or board binding (§3.8).

**Two authorities, kept apart.** The board's adapter authorises board facts: target, run, ancestry and publication. The host keeps its **independent machine-local context policy**: local clearance, builder allowlist, payload mode and enforcement key, from local configuration and the authenticated transport, never from a board record; with its fail-closed defaults and ordering, before launch and before restore-for-cleanup, and with absence distinguished from a read error. Denial tests are bound to the production path across the converters.

Provider success means exactly what it means today: Claude's exact acknowledgement and deferral; Codex's text directives; Muse's accepted adapter behaviour (unregistered).

### 2.4 The gate: three lanes

Figure: [M-SH4 SH1 gate deployment](../diagrams/sh1-gate-deployment.puml) shows
lane ownership, evidence provenance, the bounded prototype and release gates.

- **Lane 1, projection.** Public module CI on GitHub-hosted macOS and Ubuntu checks out the private board at one pinned commit. The owner accepted board code excerpts in public CI logs on 2026-10-10. The unchanged task-board tests run against the candidate's production sources projected at their original import path in a disposable pinned board checkout; only declared deterministic import and type adapters; test bytes, fixtures and goldens unchanged and hashed; discovered, executed, skipped and tagged-test inventories published; the guard tests' targets reached versus expected reported. This supersedes the older private/self-hosted lane 1 layout in contract row P9.
- **Lane 2, consumer.** The release gate is the board's private consumer workflow on GitHub-hosted Ubuntu: a real board build against the **immutable public module tag** with no candidate replacement, exercising the converters, the daemon composition and the command-line behaviour; the board's whole CI (configuration, board, vendor drift, build and document guards, all CLI packages, server and regression scenarios), not only the session subsets. The current public hosted prototype uses a file GOPROXY and runs board build/vet; it does not establish this full release gate.
- **Lane 3, module.** Public module CI on GitHub-hosted macOS and Ubuntu runs the module's own vet, race, build and declared macOS and Linux conformance. Current generic tooling and invented-fixture checks establish tooling behavior, not extracted-module conformance.
- **Mutations.** Deliberate mutations of the projected code, the real converters and each relevant guard path must each fail a named test.
- **F**, the first landing, proves a representative final graph on lanes 1 and 2 before the neutral interfaces are committed; Z and C1 rerun all three lanes against the released candidate. If F fails, SH1 stops at the mechanical compatibility milestone and the board keeps its dependencies.
- **Not evidence of more than it is:** existing opt-in smoke tests (MSP, schema export, PTY) and tagged mutation lanes keep their opt-in status; SH1's suites say nothing about the platform canons of §3.

The hosted prototype, its measured scope and its failures are recorded in [`docs/sh1-gate.md`](../docs/sh1-gate.md). Neither that tooling nor this extraction plan establishes runtime migration. No tests or builds run on the development host. Public lane artifacts contain only the allowlisted reports described in the gate guide; private checkouts and raw logs are not uploaded as lane artifacts. The accepted public-log exposure is separate from artifact publication.

### 2.5 Order of work

1. Confirm the graph and composition choice (§2.2).
2. **F**: prototype the three-lane proof; record the board commit, host tree, projection generator, helper, test and fixture hashes, module graph and environment; F stays provisional until the copy snapshot closes.
3. Schedule the freeze with the board's arbiter, after the Muse work currently landing in the package (expected around 13–14 October); pin the accepted source.
4. **O**: the snapshot copy; provenance review and leak scan before the first public push.
5. **G** and **X** (in parallel only on disjoint, frozen contracts), then **Y**.
6. **Z**: dependency exit, immutable tag, no local replacements; rerun the lanes.
7. Consumer validation on the tag; **C1** after the calendar gate.

After O the old source is frozen: each change in flight gets its accepted commit, destination commit, one named forward-port integrator and a validation identity in the freeze manifest; active reviews finish on their original base; after every accepted forward port the baseline is regenerated and the affected evidence rerun. No second runtime is authored in parallel.

### 2.6 Cutover (C1)

Figure: [M-SH4 gate and operator cutover](../diagrams/sh1-gate-deployment.puml)
includes quiescence refusal and schema-safe rollback.

- Not before the board's conference window ends (19 October).
- **Quiescence for the whole transaction.** Before stopping the daemon, the operator closes every entry path for the affected boards: the wrappers' automatic daemon start, scheduled and recovery starts, active board runners and operator entry points, using the host's existing maintenance controls. Exclusive state ownership is held and re-checked through snapshot, binary swap, start and health verification. If quiescence cannot be established, cutover is deferred.
- A paired release manifest names the board build, the module tag, the wire, goal and storage schema versions, and the permitted reader and writer pairs; the board build stamp stays in the daemon version, the host identity is reported separately.
- The operator performs the swap inside a maintenance slot; the replacement guard stays; the system reopens only with the paired identity and one verified writer.
- Disposable acceptance covers late starts by old clients and all four old/new wrapper and daemon pairings.
- Rollback opens mutable state only after proving the old reader accepts every retained schema, and never erases later records by restoring an old snapshot.

## 3. Phase SH2+: platform canons

Each item below is an **activation gate**: it is designed and tested before it is switched on, and SH1's unchanged tests are not evidence for it.

### 3.1 Service account, state and capabilities

Figures: [M-SH1 process ownership](../diagrams/containers.puml) and
[M-SH4 storage ownership](../diagrams/storage-layout.puml) show service-private
state, agent-side transports and the authenticated socket boundary.

- The host moves to its own service account `swarma-session-host` (home `/opt/swarma/services/swarma-session-host`) only after the direct file APIs (notice inbox writes, context-state reads by the board) are replaced by authenticated calls, and only together with agent-account session runners (§3.2). Running harnesses as the service account is not a boundary.
- The socket directory `/opt/swarma/run/session-host/` has published permissions that let authorised principals connect while the data stays private. Connections follow broker §4.1: kernel peer UID, server identity check, close-on-exec, one role per connection; managed principals by `(uid, generation)`.
- Account generations are fresh random 128-bit values (R-GEN). Active managed generations are checked against the user manager's root-owned ledger, by default `/opt/swarma/lib/user-manager/ledger.json`. Readability is not authority; a malformed, unreadable, ambiguous or retired record is not an absent identity and cannot authorise a fallback.
- **Ledger loss and restore (F-17/R-GEN).** Follow the user manager's state-loss/restore procedure (§§4.1, 4.5–4.6, §14) and the dispatcher's (§§4.6, 7.1, 11). The shared operator restore fence closes every ledger consumer, including the host, before restored bytes become visible. The host durably records the highest ledger store epoch it has observed and refuses a lower epoch until operator reconciliation. A reader's epoch floor helps only while its evidence survives outside the restored failure domain; it does not replace the restore fence or coverage proof, nor detect an internally consistent whole-machine rollback. Lost or incomplete evidence is unknown, not absence, and cannot reopen delivery or successor admission. A retired generation MUST NOT be revived by restore.
- **Service egress (F-24/R-SVC-EGRESS).** The host service account has deny-all network egress, including loopback TCP; platform calls use Unix sockets. The user manager's §8 owns the v1 service-profile row, parent-anchor checks, resolver sandbox qualification, start-time verification and scheduled re-verification with its published owner. Per-UID packet denial alone does not prove DNS denial through a system resolver. Host readiness requires the verified deny-all row, parent anchor and qualified resolver restriction. A missing or invalid row drops readiness and raises an incident; recovery controls follow F-15 and process response follows R-FATE, without interpreting missing policy as permission. Periodic checks leave a detection window; continuous enforcement is not claimed. Service-profile and per-OS evidence remain activation gates (§7).
- **Credential renewal (F-14/R-REBIND).** Standing-session lifetime does not extend credential authority. Follow dispatcher E-rebind (§§3.4–3.5, 5.2, 7.1–7.2) and broker binding replacement (§§7.1–7.3, 8.3, 8.5): the same `(uid, generation)` is retained, the leaf is re-signed under current chains, the old binding has a removal tombstone, and each rebind uses a fresh `revocation.check` with context `prepare`. The renewing principal must already be in the chain; authority never widens and the account is not re-created. Pending or expired renewal keeps dependent readiness, leases and restarts closed; an env-channel token already issued is not withdrawn by replacing the binding. A revoked envelope or key requires an explicit operator decision, never automatic renewal.
- **Time confidence (F-21/R-TIME).** The installer requires an authenticated or operator-controlled time source, for example NTS, as specified by the user manager's installer and the dispatcher/broker time contracts. Their durable time watermark checks expiry at `max(now, watermark)` and `not_before` at `min(now, watermark)`; it never advances on untrusted time and its reset is an explicit operator act. Clock steps can only narrow authority. Expiry-driven home deletion or binding removal also needs boot-inclusive monotonic elapsed time of at least the lifetime; it is not triggered by the host's liveness deadline. The host follows the shared time-confidence hold policy: untrusted time cannot justify successor admission or revive expired session authority; explicit authorised stops remain available.
- **Cross-account workspace (F-16).** A child in another isolated account gets a standalone repository with its own private `.git`, configuration and hooks, under the launcher's workspace materialisation contract. No linked `git worktree` or inherited administrative backlink crosses that boundary. Same-account worktrees and today's board workflow stay unchanged; this host adds no workspace-transfer authority.
- Capabilities: issued by the host to a logical session and its execution, with audience, actions, expiry and revocation; parent and child relationships are authoritative host records; replacement rotates execution-scoped authority; controller epochs are reconciled before effects. The legacy global token is refused after activation. Trusted audit and transcript evidence lives at the service boundary, not in agent-writable files.
- Standing sessions held by the host — the orchestrator and standing team members, including other orchestrators and standing agents — MUST have no preset lifetime or runtime limit (§3.5). Capability expiry bounds authority; it does not impose a standing-session lifetime. Operator maintenance uses the dispatcher's bounded maintenance stop (R-MAINT, §3.5).

### 3.2 Holders under agent accounts and the process tree

Figures: [M-SH1 containers](../diagrams/containers.puml),
[M-SH1 components](../diagrams/components.puml) and
[M-SH2 recovery sequence](../diagrams/session-recovery.puml) show runner
ownership, re-authentication, confirmed absence and privileged launch ordering.

Before session runners are activated, the ownership table is frozen:

| Concern | Owner |
|---|---|
| the PTY master or app-server transport | the session runner `swarma-session-runner`, one process per session, started only by the dispatcher through `swarma-user-launcher` under the agent account |
| the credential lease client (renewals for Codex app-server) | the same agent-side session runner, for the whole execution |
| durable metadata, logs, receipts | the host service |
| admission, allocation and execution records, cleanup | the dispatcher |
| stop | the host requests `host_stop`; the dispatcher owns the journaled E-stop path through the user manager's `exec.stop` (confirmed process absence required) |

The runner binary belongs to the session-host repository. Codex has a private app-server per session under the agent UID, with no shared service-account app-server or credential pool. The runner renews its execution's credential lease; only the broker's auth owner refreshes the rotating Codex account. Broker-mode Codex uses an external access token and account ID, without copying `auth.json` into the launch home or using `codex exec`. Supported provider releases still require qualification (§7). The broker's default renewal deadline is 8 seconds; a timeout fails the turn, with no late reply or native-auth fallback. Claude setup-token has no refresh.

D-EXECUTOR makes agent-side `curator-run` the first broker consumer. The board runner and session host ask the dispatcher rather than embedding a broker client. In the versioned one-shot mode, `curator-run` under the agent account MUST only verify and execute an immutable plan composed by the caller; it MUST NOT compose or resolve profiles and has no network access except the broker. It shares one launch library with `swarma-session-runner`. A separate minimal executor binary is a later option, not the current choice; the exact versioned intake remains owned by the launcher specification.

- The freshly started runner opens a fresh authenticated channel to the host after the UID change, compatible with the launcher's descriptor rules (plan on descriptor 3, nothing else inherited). Registration for the admitted execution in the current controller epoch and readiness MUST precede delivery; a replacement host re-authenticates and re-adopts a live runner only after reconciliation.
- The root launcher validates the active owned generation, approved executor path/digest and home, resets descriptors and environment, and writes the atomic start receipt before clearing groups, setting GID/UID and executing. Descriptor 3 carries only the plan; no broker connection is inherited. The runner opens its broker connection after the UID change; the authenticated host channel must be established without inheriting another descriptor (§7).
- **Process fate (F-18/R-FATE).** The user manager’s §7 owns the per-process-class **fate table**, beside R-FD3. Runner roots and descendants, host-service replacement, dispatcher supervisors and broker children follow its published service-manager/kill-domain rows, stop/crash/restart fates, observation rules and R-MAINT affected execution sets. A live runner is re-adopted only when the backend promises and qualifies survival; `setsid` and UID change alone do not prove containment or survival. Every supervisor write is fenced by the durable dispatcher daemon-instance epoch; a supervisor left over from a replaced daemon is never a live adopted writer. Platform services and their service managers end executions only through `exec.stop`, retirement or operator emergency stop; OS failures such as OOM remain possible and are recorded as `lost`. Publication and per-platform qualification are required before claiming runner survival (§7).
- Dispatcher recovery distinguishes an absent controller, a live authenticated session runner, an absent execution and unknown evidence; only confirmed absence permits replacement or cleanup. Controller-only re-adoption keeps the execution id; a new exec gets a new one.
- Start, readiness and exit receipts are acknowledged and replayable; capacity release follows confirmed absence.

### 3.3 Backend activation with the dispatcher and the user manager

Enabled only with jointly pinned host, dispatcher and user-manager versions:

- `host_start` in `dispatch-op/1` (R-START/R-SESSION), accepted only from the configured session-host service principal, carrying `(logical_session_id, execution_id, controller_epoch, op_id)` and referring to a durable dispatcher-owned **host admission permit** scoped to one `logical_session_id`;
- the permit is created at the owner's admitted start with the session-host executor. The owner's **session lifecycle authorisation** explicitly authorises successors, if present, and is retained with the admitted input and session record; its member format is unpublished. A one-start request without it authorises no successor;
- every physical start keeps its own `(execution_id, controller_epoch, op_id)` claim, fence and same-effect/root-receipt interlock. The approved runner, unchanged canonical plan digest and active account generation remain part of the admitted launch path;
- `host_stop` in `dispatch-op/1` (R-SESSION) is accepted only from the configured session-host principal and only for an execution under its permit. It uses the dispatcher's journaled E-stop path (`exec.stop`, confirmed absence), records the stop as host-requested, is distinct from owner `cancel`, and confers no start authority;
- identities kept distinct: account generation, execution identity and the backend's controller epoch;
- the interactive and resume subset negotiated explicitly (dispatcher v0 refuses them);
- negative qualification cases: wrong principal, wrong session or execution, replay, one-start request used for a successor, retired owner, revoked/expired chain, altered plan or executor, withdrawn/unresolved permit, prior claim not absent, cancelled claim, ambiguous start, host failure and crash on each acknowledgement.

**Successor admission (F-20/R-SESSION).** Only the owner's session lifecycle authorisation admits a successor `host_start`, idempotent per `(logical_session_id, op_id)`: replay returns the same successor, never a second. The previous claim must be confirmed absent first. Each use keeps the same allocation and generation and reuses the stored plan verbatim; the host supplies no plan bytes. Pin or posture divergence needs a new owner admission. The original owner must remain active, its chain unrevoked and unexpired; each authority-creating use passes fresh delayed-work checks and `revocation.check` with context `start`, with distinct effect IDs. A retired one-shot orchestrator's sessions are not restartable. The host acquires no caller chain, spawn grant or lease grant; the binding retains the original owner's caller dependency and ancestry. Successors stay within the owner's grant window and allocation lifetime. For ordinary work the authorisation has a cumulative or first-start-anchored restart budget, so successors cannot reset runtime windows; standing executions retain their no-runtime-limit rule. The budget format remains unpublished (§7).

The host prepares durable session intent and a closed delivery fence before `host_start`. Only the dispatcher invokes `swarma-user-launcher`, through its existing claim, effect and root-receipt path. The session-scoped permit spans physical starts, with the per-start fence interlock. A fence change withdraws the affected start's authority or sends it to reconciliation; unknown permit state keeps successors closed. Owner `cancel` or `retire`, chain revocation or expiry, the operator's signed standing-retirement record, and R-MAINT withdraw the permit durably and serialise with successor admission under the dispatcher's state lock. No autonomous repeat follows withdrawal. Controller-only re-adoption retains `execution_id` and needs no permit action.

Cancellation before the privileged call prevents the call; after an ambiguous call the dispatcher resolves the same effect and receipt and stops the execution if required, never launching twice. Host failure keeps the dispatcher claim and closed fence until reconciliation. Current-epoch runner registration and readiness precede delivery (§3.2–3.5). These semantics refine F-4; joint publication and qualification remain §7 question 2.

**Dispatcher controls while admission is closed (F-15).** The dispatcher is an unprivileged Unix-socket service from its first protected-launch slice (F-5), using broker §4.1 and §13 connection/framing rules. The host verifies server identity and authenticates as its configured service principal; the dispatcher verifies caller identity, active generation where applicable and request signatures. After exclusive state ownership, schema compatibility, journal replay and valid authentication/indexes, authorised status, list, observation, reconcile and already-authorised current-execution-bound stop remain reachable even when forward admission is closed. Observation of an old effect needs no new launch grant. An absent dispatcher has no command fallback; unusable shared state cannot be bypassed.

Before forward admission every non-terminal record is classified. Unresolved records retain their generations, allocations and quota/capacity claims; forward actions on them remain closed. Unrelated work is admissible only with complete accounting and proof that possible unresolved effects are confined to those retained claims and identities. If affected identities, accounting, shared authorisation or shared-state integrity cannot be bounded, forward admission stays closed node-wide. Maintenance, restore fencing and the existing fail-closed hold gate remain global barriers. No claim is released on unknown evidence, and no live supervisor is adopted. The operation-availability matrix and confinement proof are owned by the dispatcher (§§7–9, 11); this host adds no alternate launch path. Queueing, events, long waits and advanced integrations remain later capabilities on the same transport.

D-WIRE-NAMES requires versioned platform wire identifiers to carry their repository name: `swarma-<component> <object>/<n>`. Examples are `swarma-dispatcher request/1`, `swarma-credential-broker grant/1` and `swarma-credential-broker revocation/1`. The rule also covers `dispatch-op/1` and user-manager request kinds, whose exact objects and grammars must be published by their owners. Nothing has shipped, so no migration is required. Naming does not supply missing signature grammar or activate grant verification; user-manager v0/v1 use dispatcher-to-root caller/role policy, and caller-bound grant verification is a v2 gate.

D-PATH fixes the launch-scoped PATH in the digested plan: (1) this launch's protected tool directory (CIP-0007); (2) the admitted base PATH in its existing order; (3) one protected manager dispatcher directory, through which project commands run after checks. A repository `.agents/bin` MUST NOT be executed in a managed launch. The resolved PATH is recorded in the plan and covered by its digest.

### 3.4 Effects, fences and admission

Figures: [M-SH3 approval and receipt states](../diagrams/approval-receipt-states.puml)
and [M-SH4 storage ownership](../diagrams/storage-layout.puml) distinguish durable
evidence, unresolved outcomes and refusal. State labels illustrate the prose;
they do not publish the pending wire/state schema.

- **Effect matrix.** Every effect — enqueue, deliver, steer, interrupt, goal and control calls, attach, start, wake, retry, probe, session runner re-adoption — has its fence and epoch rule and idempotent `op_id` behaviour. A closed fence may queue but not deliver; only the operation holding the fence may restart; an explicit stop stays stopped; after a controller failure nothing writes to session runners before reconciliation.
- **One node admission authority** relates the dispatcher's capacity and execution claims, the host's fences and the session-scoped host admission permit and each physical start’s claim/fence held through start, with reconciled authority for verified re-adoption. Every starter is inventoried (board runners, manual and alias launchers, restart and recovery consumers, probes, retries, fallback legs, mixed-version clients); an unsupported participant blocks shared mutation. A fence change withdraws the affected physical start’s authority or sends it to reconciliation; host failure retains the dispatcher claim and closed fence. Unknown permits stay owned; unreadable state is closed; ordinary restart closure is separate from an operator's update lock, which only the operator clears.

**Wait/lock rule (F-28/R-LOCK).** The host MUST **never hold its state lock across a synchronous call to another platform service**. This includes its fence lock across `host_start`, `host_stop`, permit withdrawal/reconciliation and runner calls. Under the state lock it journals immutable intent and a durable per-record in-flight marker, then releases the lock, calls the receiver, and reacquires the lock to record/revalidate the result. Ownership is serialised by that marker, not by holding the global lock across I/O. The effect identity, canonical arguments and caller provenance are fixed at intent; authority is freshly checked at authority-creating uses, never cached from a prior verdict.

Each call is bound to its record and current daemon-instance/controller epochs without assuming a new wire member. Late results are revalidated against current fences and versions: stale results may resolve historical evidence, but never reopen delivery/admission or resurrect cancelled starts. Cancellation before issue prevents the call; cancellation or fence withdrawal during a call records intent promptly and observes the in-flight marker, so the post-call handler stops rather than proceeds. The host releases the fence lock before sending withdrawal to the dispatcher; the dispatcher reconciles under its claim/receipt interlock. Claims for possibly executed work remain retained until confirmed absence. Outbound I/O and in-flight work are bounded per service and serialised per record, leaving unrelated records and authorised status/stop controls reachable. Bounds and marker schema would need publication (§7 question 2).

The host's wait/lock graph is [M-SH5](../diagrams/host-wait-lock.puml): state lock → local journal/record transition only; unlocked in-flight operation → dispatcher or runner call → unlocked result handler → state-lock revalidation. There is no state-lock-held edge to another service, so the lock-wait graph is acyclic even though host and runner calls go both ways. Singleton/process-ownership locks have different lifetimes and are exempt; the rule does not release R-FD3/root generation exclusion or refresh-writer ownership. Qualification would need slow dependency with concurrent status/stop, mid-call fence withdrawal, cancellation and stale completion after replacement.

**Receipts and approvals (SH2+ target, schema pending).** Delivery records distinguish `accepted`, `queued`, `dispatched`, `applied`, `refused`, `expired` and `unknown`; no acknowledgement is emitted without a durable write. After a host or runner failure between dispatch and application, reconcile the input marker against an intact transcript before reopening the fence. A matching marker proves applied; confirmed absence in an intact transcript permits redelivery; unresolved evidence stays unknown. An unknown approval answer or control effect MUST NOT be replayed automatically. Historical approvals become invalid on restart.

A pending native interaction binds its execution, turn, native request digest, options and expiry. An authorised approval must target that pending interaction and the current execution/epoch, match the digest and remain unexpired. Compare-and-swap accepts the first valid resolution; an ordinary user turn is not an approval. Closing the fence invalidates pending interactions. Automatic approval requires an explicit allowlist policy. These are [architecture receipt and interaction requirements](https://github.com/relux-works/wiki/blob/2e30edbd54977b306ab53497663fcd94a5ca6cb6/session-host/architecture.ru.md#L376) to carry into a dedicated module wire/state schema before activation, not a claim that SH1 implements them; this revision invents no wire fields or policy values for that schema.

### 3.5 Park, restart, emergency stop, board completion

Figures: [M-SH2 recovery sequence](../diagrams/session-recovery.puml) and
[M-SH3 independent lifecycle views](../diagrams/session-goal-wake-states.puml)
keep controller recovery, execution absence, park and optional board completion
separate.

- **Standing-session lifetime (owner decision, 2026-10-12).** A standing session held by the host MUST have no preset lifetime or runtime limit, and its executions MUST have no preset runtime maximum. It ends only by an explicit stop, by operator action (including emergency stop), or by its own exit. Park and restart MUST NOT end the standing session: park holds delivery closed, while controlled restart replaces an execution within the same session. The dispatcher's 12-hour maximum execution runtime (§3.5 of its specification) applies only to non-standing executions. This decision supersedes R3; it does not activate the SH2+ backend or publish a runtime-policy wire format.
- **Operator maintenance (R-MAINT, dispatcher §11).** Standing sessions participate in the dispatcher's bounded maintenance attempt: close new admission, inventory existing executions, then stop selected executions with explicit operator authority through the dispatcher and `exec.stop`. Confirmed process absence and reconciliation are required before any change of mode, mutable-state migration or binary swap. The maintenance deadline bounds the attempt, never the session's or standing execution's lifetime. Unknown or refused stop, or incomplete reconciliation, defers maintenance. Standing allocations and identities remain; there is no live dispatcher adoption or automatic expiry. Host controller-only re-adoption (§3.2) is a separate recovery operation. R-MAINT withdraws lifecycle authority; any intended restart remains subject to operator review and fresh owner admission with an admitted plan/transport. Recovery controls remain reachable while forward admission is closed (F-15, §3.3); unsupported rows stay visibly stopped.
- **Park** is a durable host operation that holds delivery closed. If a board goal is bound, the optional board client acknowledges a committed goal revision carrying the handoff reference (compare-and-swap); that board barrier is not a prerequisite for a goal-free session. The note attempt is bounded, with an explicitly marked snapshot fallback; a timeout never proves drain. For a wedged execution, the host requests `host_stop` under its permit and retains the fence and claims until confirmed absence; that stop grants no successor authority. The goal-free park receipt/state schema must be published before activation (§7).
- **Restart** follows the architecture's transaction under a closed delivery fence: drain external effects and child tools, request `host_stop` from the dispatcher under the session-scoped permit, through its journaled E-stop path and the user manager’s `exec.stop`, confirm process absence, check binding, posture, authentication, grants and runtime, then request a successor execution and fresh agent-side runner through `host_start`, solely under the owner’s session lifecycle authorisation and session-scoped host admission permit (§3.3). `exec.stop` follows the single termination scope in the user manager’s §§4.4 and 4.7 (F-13): while one execution per generation holds, it signals every process of the generation’s UID (TERM, grace period, KILL), stops its per-user service domain where present, and verifies account-wide absence over a quiet period. Receipt/process identity authorises the target; it does not limit the signalled set to one process session or a wall-clock receipt window. Account-wide signalling requires the generation’s latest execution receipt; a successful stop replays without signalling, and an old receipt never stops a later execution. Future concurrent executions would need exact per-execution containment before activation. The stop is harness-agnostic and proves process absence only; unkillable processes remain unknown. Runner fate and the affected set on service replacement follow R-FATE (§3.2), never deployment defaults. The fresh runner checks the harness's native-session lock under the closed fence and reports the result to the host before harness initialisation or native resume. Only confirmed native-lock absence permits that step; an unknown result keeps the session parked with delivery closed. Then initialise or resume, reconcile unknown delivery and reopen only after current-epoch registration, readiness and reconciliation. The host does not signal the processes directly. Unknown process evidence cannot release capacity or authorise replacement.
- **Native resume (D-RESUME).** On the same machine, a non-breaking harness semver change is allowed without an extra qualification row: the same major, and for 0.x the same minor. A different major or 0.x minor MUST be refused. Keep the same runtime family, managed home (checkout × profile × environment), profile pin and intact transcript digest, exclusive native-session lock, ready authentication and current grants. Preserve the admitted transport (`tty` with PTY preservation, or `hosted-stream:<adapter>`); no transport conversion is inferred. If the harness fails native resume, report `resume-failed` to the caller and let the caller decide the next action. There is no automatic fallback machinery, and portable seed is not part of this resume path. A per-harness note records vendors known not to follow semver. This replaces the unresolved version choice in contract row P8; SH1 extraction itself still preserves existing behavior.
- **Recovery matrix:** provider × owner kind (primary, owner-run) × source record (hosted, legacy) × recovery mode, each row with its fresh plan and credential supply, authority, home and native identity and readiness checks; unsupported rows stay visibly parked or refused; legacy ambient environments are never replayed into the new account.
- **Board completion**, when an optional board client is bound, is a separate barrier: an execution's exit never satisfies the board's `wait`; exit receipts are durable and replayable.
- **Emergency stop:** an operator command with defined scope and authority, a snapshot of any bound board first, `host_stop` under the session permit through the journaled E-stop path, fence and confirmed-absence receipts, and explicit recovery rather than automatic restart. It is host-requested, distinct from owner `cancel`, and confers no start authority. Sessions without a board do not require a board snapshot.

### 3.6 Existing probes

SH1 preserves the existing adapter paths as they are: Muse PTY's `--version` preflight, the MSP initialize handshake and its identity check, and the last-known-good fallback start. They do not meet the platform's isolation rule. Before platform activation each startup, probe and separately admitted fallback path gets an adopted qualification policy: approved artifact identity before exec, isolated state or central qualification, a registered attempt, exact result binding and confirmed teardown; each fallback leg gets its own admitted execution. Without that evidence the new platform route refuses. Native resume follows §3.5 and D-RESUME: a non-breaking semver change does not require an extra qualification row or successful version-change probe, and resume failure does not start a fallback leg.

### 3.7 Wake and liveness

No hosted agent re-arms its own wake-up; there is no `wait` for the model to forget.

| Responsibility | Owner |
|---|---|
| receive mail for an agent identity and hold it durably | the bridge (carrier provider and Mailbox); it enqueues into the host's per-session inbox and counts delivery only after the host's durable write |
| decide when to deliver (the doorbell) | the host, the only writer of a session's input; it delivers at a turn boundary |
| report the turn boundary, idleness and heartbeats | the session runner (§3.2), through the adapter's readiness grammar: Claude PTY or stream output, Codex app-server turn events; an unknown readiness parks rather than guesses |
| detect a wedged session | the host: pending inbox items, and the session neither idle nor progressing past its deadline |
| recover | the host's restart transaction (§3.5) under a new controller epoch: `host_stop`, confirmed prior-claim absence, then successor `host_start` only under the owner’s session lifecycle authorisation and session-scoped permit, then redelivery, idempotent by sender and message id; the dispatcher for admitted starts, execution stop, absence and cleanup (§3.2–3.5) |
| never woken | an explicitly stopped or parked session, a closed fence (it queues, it does not deliver), a session that needs re-authentication (parked) |

A wake is an effect in the §3.4 matrix: fenced, epoch-checked, with an idempotent `op_id`. The binding from an agent identity to a session names the session, the execution and the epoch, so a wake after a restart never reaches a stale session. Claude Code and Codex are covered; Muse after its qualification (§3.6).

Liveness deadlines use monotonic elapsed time within a boot; after reboot unknown elapsed evidence cannot justify a guessed deadline. They detect a wedge; they MUST NOT impose a lifetime or runtime limit on a standing session. Recovery by park or controlled restart preserves that session (§3.5); only an explicit stop, operator action or its own exit ends it. Time confidence follows R-TIME (§3.1); a clock step cannot revive authority or turn wedge detection into expiry-driven destruction. Maintenance uses R-MAINT (§3.5); its deadline bounds the attempt, with recovery controls reachable while forward admission is closed (F-15, §3.3).

Until SH2+ is active, interim rails live outside this module: a per-identity watcher that holds the wait itself and injects one fixed line only into an idle session with an empty input line and never into one the operator stopped; and, for sessions in today's daemon, its notice path, which delivers only while a client is attached (§1.1).

Figure: [M-SH3 session/goal/wake states](../diagrams/session-goal-wake-states.puml)
follows this table and keeps session, execution, goal acknowledgement and optional
board waiting states separate. Unspecified board transitions and pending schemas
are noted rather than supplied by the figure.

### 3.8 Goals across accounts

Figures: [M-SH1 cross-account components](../diagrams/components.puml) and
[M-SH3 goal command/acknowledgement view](../diagrams/session-goal-wake-states.puml)
show caller authorship, optional board ownership and host-to-runner delivery.

**D-GOALS: variant (b) now.** Only the board/orchestrator for board goals, or the launching caller at start through the launcher, sets a session's goal. The agent account has no goal capability on the host socket. This keeps goal-authoring authority with the initiating caller while the separate agent namespace and its bounds await SH2+ design.

The goal interface is generic and optional. The host MUST work without a board, goals, our orchestrator or its toolchain; goal absence MUST NOT block launch, attach, delivery, wake, stop or recovery. The board's ledger, ancestry, outcome proposals and completion barrier belong to its optional adapter (§2.2–2.3), not to the host's session lifecycle. SH1 retains today's board adapter behavior through its compatibility seam; it does not make that adapter a mandatory dependency of the extracted module.

**Unspecified goal surfaces (§7).** The goal-client interface on the host socket, including its actions and capability shape, is unpublished; the flow below states authority and ordering, not an API. The source of a per-session nudge or wake budget, who sets it and the default when no board is bound are unspecified. There is no specified path for an agent to propose "done" when no board is bound. With a board, an agent outcome is only a proposal to the board owner. D-GOALS gives the agent account no goal capability: any future proposal path MUST NOT mutate a goal.

When a goal is present, the authoring caller, the host and the agent run under separate accounts. The optional board client runs under the orchestrator's account or its service account; the host under `swarma-session-host`; the runner and harness under `worker-<label>`. The SH2+ flow is:

1. An authorised launching caller passes an initial goal through the launcher, or the orchestrator sets or updates a board goal through the optional board binding.
2. The initiating caller's goal client asks the host to apply revision N to session S. The board binding is one such client. The host identifies the caller by kernel peer credentials (UID and active generation) and requires its session-scoped capability with the applicable goal actions, audience, expiry and revocation. S's fence must be open. An agent-side executor carrying a caller-composed plan acquires no goal-authoring capability.
3. The host records the revision durably. At the next turn boundary it hands the revision to the runner of the current epoch, which writes it into the harness input (§3.2, §3.7) and returns the harness's acknowledgement.
4. The host answers that client: applied, deferred, or rolled back with the joined error, as in today's goal seam. The optional board adapter keeps its ledger transaction and exact-revision acknowledgement semantics.

The runner accepts input only from the host. Its fresh authenticated channel and registration must establish that it belongs to S's admitted execution in the current controller epoch before readiness and delivery; the host talks only to that registered runner. The proof format remains unpublished (§7 question 2). Revoking the initiating principal's grant or session capability stops its goal operations at once; stopping the execution stays with the dispatcher through the user manager's `exec.stop`.

**Deferred SH2+ item (G2).** The final choice is (a) agent self-goals in a separate namespace, with its authority and bounds designed separately, or (b) retaining caller/board authorship only. No agent self-goal API is enabled now. Agent-reported board outcomes remain proposals to the board owner, not agent-authored changes to a board goal. This later choice does not block the goal-free host.

In SH2+ the module also ships the host service binary, and the optional board client uses this socket instead of compiling the session manager in.

## 4. Estimates

Stated velocity (2026-10-09): a clean accepted change takes 3–4 hours from producer start to landing; one rework or base movement makes it 10–12 hours; two runs at a time on the shared host. These are serial sums, not commitments; the landing time is already inside the rate.

| Work | All clean | One rework each |
|---|---:|---:|
| SH1 module landings F, O, G, X, Y, Z | 18–24 landing-hours | 60–72 landing-hours |
| C1 consumer cutover | plus a maintenance slot after 19 October | |
| SH2+ (15–25 changes across host, dispatcher and user manager) | 45–100 landing-hours | 150–300 landing-hours |

With continuous agent coverage, SH1 fits in 3–4 calendar days clean and about 5–7 with rework, if F passes; SH2+ in about 1.5–2.5 weeks clean. The critical path is the jointly versioned host, dispatcher and user-manager activation (§3.2–3.3) and the consumer acceptance, not the copy. CI times are measured at F and added only where they are not already inside the rate.

## 5. Dispositions

| Item | Disposition |
|---|---|
| frozen-plan gap R1 (resize) | §2.3 |
| D1 (restart closure) | §3.4 |
| D2 (probe isolation) | §3.6 |
| cutover through the private fork | superseded: the board consumes a tagged public module through C1 |
| operator-only deployment and update-lock clearance | §2.6, §3.4 |
| CI capacity | §2.4: public hosted projection with a private board checkout and accepted public-log exposure; private full consumer CI for the release gate; public module conformance |
| calendar gate | 19 October (owner decision 2026-10-09) |
| r3 review F1–F12 | §1.1 and §2.1 (F1), §2.3 (F1, F2), §2.4 (F3), §2.2 and §2.5 (F4), §2.6 (F5), §3.3 (F6), §3.2 (F7), §3.1 (F8), §3.4 (F9), §3.5 (F10), §3.6 (F11), §4 (F12) |
| R-SH1 audit A07.1–A07.10 | Service/runner and private app-server (§1, §3.1–3.2); receipt/approval schema pending (§3.4, §7); dispatcher-owned stop (§3.5); D-RESUME (§3.5–3.6); optional goals and deferred self-goals (§3.8); wake (§3.7); lane ownership and bounded tooling evidence (§2.4); tagged public module and operator cutover (§2.2, §2.5–2.6) |
| Historical diagram inventory | Audit B1 found no session-host diagram files or inline diagrams at its pinned revision. External wiki/book figures belong to R-WA1/B-F2. |
| Accepted review-findings wave (2026-10-12) | F-4/R-START: §3.2–3.4, §3.5, §3.7–3.8, §7; F-2/R-MAINT: §3.1, §3.5, §3.7, §7; F-5: §3.3. M-SH1–M-SH4 target ownership, recovery, lifecycle and storage views follow these contracts. |
| Accepted cycle-2 review (2026-10-12) | F-20/R-SESSION: §§3.3–3.5, 3.7, §7.2; F-15: §§3.3, 3.5, 3.7; F-18/R-FATE: §§3.2, 3.5, §7.11; F-13: §3.5; F-24/R-SVC-EGRESS: §3.1, §7.12; F-28/R-LOCK: §3.4, §7.2 and M-SH5; F-14/R-REBIND, F-17/R-GEN, F-21/R-TIME and F-16: §3.1 (time also §3.7). M-SH1–M-SH4 figures follow the revised lifecycle and boundaries. |
| Module diagrams | [R-SH2 figure index](../diagrams/README.md): M-SH1 process/component ownership, M-SH2 recovery sequence, M-SH3 session/goal/wake and approval/receipt states, M-SH4 storage and gate deployment, M-SH5 host wait/lock graph. Their prose authority is this revision; the gate view uses §2.4's corrected lane 1 layout. Figures do not activate unpublished contracts. |

## 6. Out of scope

Terminal backends other than the per-session session runner (tmux), the `ax` backend, the three-operating-system demo, GUI and mobile distribution.

## 7. Open questions

1. Which board key replaces `BoardKey` once one host serves every board on a node (must be settled before multi-board admission and capabilities in SH2+).
2. **UNPUBLISHED — host lifecycle and runner channel (R-START/R-SESSION/R-LOCK; replaces I4).** Joint publication would need the session lifecycle authorisation member format, ordinary-work cumulative or anchored restart budget, exact `dispatch-op/1` framing, authentication/signature domain and canonical bytes for `host_start` and `host_stop`, host-requested stop recording, and the session-scoped host admission permit's durable storage, withdrawal, per-start claim/fence/effect/receipt crash ordering and refusal codes. It would also need in-flight marker/recovery formats, bounded I/O policy, stale-result epoch binding and the fresh authenticated runner/host channel's execution proof, current-epoch registration, readiness and reconnection formats. The fixed four `host_start` request values and §3.3 permit semantics are authoritative; no additional wire fields, refusal codes or values are assumed. FD3 carries the plan only, with no extra inherited descriptor. The reboot requester (host or project registry) remains unresolved; whichever is configured acts only within the owner's session lifecycle authorisation. Publication/qualification gates backend activation; launch authority is settled.
3. **OWNER DECISION NEEDED — user-manager v2 signatures (W3).** Adopt a component-specific signed-request format or a versioned shared adapter. Either must follow D-WIRE-NAMES and bind the authenticated caller. Until its owner publishes the grammar and joint activation, the host must not claim that current dispatcher-to-root caller policy is grant verification.
4. **OWNER DECISION NEEDED — Codex personal broker lane (C4).** Adopt the single auth-owner model with named supported-release qualification evidence, or defer that provider lane. The per-session app-server boundary in §3.2 is specified; release support is not proven by that design.
5. **OWNER DECISION NEEDED later — agent self-goals (G2).** Choose a separate agent namespace with designed authority/bounds, or retain caller/board authorship only. D-GOALS selects (b) for now; this question does not block sessions without goals.
6. **OWNER DECISION NEEDED — Muse subscription lane (C7).** Prioritise qualification and select a supported channel from evidence, or defer it while keeping the API-key lane separately labelled. Subscription refresh/channel support remains reserved for research.
7. **UNPUBLISHED — goal-client interface (§3.8).** What are the host-socket goal actions and capability shape? Publish the interface before activating it; the described authorisation flow supplies neither wire fields nor action names. D-GOALS keeps goal capability with authorised caller/board principals; the agent account has none.
8. **UNSPECIFIED — per-session nudge or wake budget (§3.7–3.8).** Who supplies and sets it, and what is the default when no board is bound? No budget source or default is assumed here.
9. **UNSPECIFIED — agent "done" proposal without a board (§3.8).** No such path is specified. With a board, an agent outcome is only a proposal to the board owner. Any future proposal path MUST NOT mutate a goal: D-GOALS gives the agent account no goal capability.
10. **RESOLVED — standing-session execution runtime (owner decision, 2026-10-12; supersedes R3).** Standing sessions and their executions have no preset lifetime or runtime limit (§3.5). Only explicit stop, operator action (including emergency stop) or own exit ends the session; park and restart do not. The dispatcher's 12-hour maximum applies only to non-standing executions. The cross-module runtime-policy wire format remains unpublished; its owners must publish it before SH2+ activation without reopening this lifetime decision. The bounded maintenance stop follows R-MAINT (§3.5, dispatcher §11).

11. **UNPUBLISHED — process fate (F-18/R-FATE).** User manager §7 would need the process-class fate table beside R-FD3, service-manager/kill-domain configuration, affected execution sets for each service stop/crash/replacement, survival promises and per-platform qualification. No survival is inferred from `setsid` or re-adoption prose; the dispatcher daemon-instance writer fence remains mandatory (§3.2).
12. **UNPUBLISHED — service egress and time confidence (F-24/R-SVC-EGRESS, F-21/R-TIME).** User manager §8 and installer would need the v1 service-profile lifecycle, deny-all row, DNS/resolver sandbox evidence, schedule/owner and lapse response, plus the shared time-confidence policy, tolerances, boot confidence and durable elapsed-time evidence. Dispatcher and broker own their watermarks. Host readiness is gated on the published/qualified service policy (§3.1).

Before SH2+ activation, the module must also publish its receipt/approval wire and state schema (§3.4), goal-free park receipts and the runner's native-lock report format (§3.5), and optional goal command/ack schema (§3.8). These are specification publication work, not invented values or evidence of current implementation. Native-resume version compatibility is settled by D-RESUME, not an open qualification-range decision.

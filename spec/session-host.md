# curator-session-host: specification

- **Status:** draft r4 (2026-10-10; §3.7 wake and liveness, §3.8 goals across accounts and the final names added the same day). Nothing has been moved yet.
- **Normative language:** MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.
- **Lineage:** continues the earlier session-host specification and the as-is extraction plan (revision 2, frozen as the reference alternative on 2026-10-05). The owner chose the as-is path on 2026-10-09: extract one to one first, then bring the host to the platform's rules. r3 closed the frozen plan's known gaps; r4 takes the architecture review of r3 (0 P0, 8 P1, 4 P2).
- **Companions:** [curator-dispatcher](https://github.com/relux-works/curator-dispatcher) (admission, allocations, executions, the session-host backend gate), [curator-host-helper](https://github.com/relux-works/curator-host-helper) (agent accounts, the launcher, `exec.stop`), [curator-credential-broker](https://github.com/relux-works/curator-credential-broker), [agent-session-bridge](https://github.com/relux-works/agent-session-bridge), the platform architecture §5.1, §5.3, §7.5.

## 1. What the host is

The session host keeps agent sessions alive in the background: it starts a harness in a given directory with a given goal, lets a person attach and detach, delivers messages into a session, and reports events. Today this is `tb-sessiond`, the session manager inside task-board (`tools/board-cli/internal/sessionmanager` in relux-works/skill-project-management).

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

The file-backed notice API stays as it is in SH1.

**Two authorities, kept apart.** The board's adapter authorises board facts: target, run, ancestry and publication. The host keeps its **independent machine-local context policy**: local clearance, builder allowlist, payload mode and enforcement key, from local configuration and the authenticated transport, never from a board record; with its fail-closed defaults and ordering, before launch and before restore-for-cleanup, and with absence distinguished from a read error. Denial tests are bound to the production path across the converters.

Provider success means exactly what it means today: Claude's exact acknowledgement and deferral; Codex's text directives; Muse's accepted adapter behaviour (unregistered).

### 2.4 The gate: three lanes

- **Lane 1, projection.** The unchanged task-board tests run against the candidate's production sources projected at their original import path in a disposable pinned board checkout; only declared deterministic import and type adapters; test bytes, fixtures and goldens unchanged and hashed; discovered, executed, skipped and tagged-test inventories published; the guard tests' targets reached versus expected reported.
- **Lane 2, consumer.** A real board build against the **immutable module tag** with no local replacement, exercising the converters, the daemon composition and the command-line behaviour; the board's whole CI (configuration, board, vendor drift, build and document guards, all CLI packages, server and regression scenarios), not only the session subsets.
- **Lane 3, module.** The module's own vet, race and declared macOS and Linux conformance.
- **Mutations.** Deliberate mutations of the projected code, the real converters and each relevant guard path must each fail a named test.
- **F**, the first landing, proves a representative final graph on lanes 1 and 2 before the neutral interfaces are committed; Z and C1 rerun all three lanes against the released candidate. If F fails, SH1 stops at the mechanical compatibility milestone and the board keeps its dependencies.
- **Not evidence of more than it is:** existing opt-in smoke tests (MSP, schema export, PTY) and tagged mutation lanes keep their opt-in status; SH1's suites say nothing about the platform canons of §3.

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

- Not before the board's conference window ends (19 October).
- **Quiescence for the whole transaction.** Before stopping the daemon, the operator closes every entry path for the affected boards: the wrappers' automatic daemon start, scheduled and recovery starts, active board runners and operator entry points, using the host's existing maintenance controls. Exclusive state ownership is held and re-checked through snapshot, binary swap, start and health verification. If quiescence cannot be established, cutover is deferred.
- A paired release manifest names the board build, the module tag, the wire, goal and storage schema versions, and the permitted reader and writer pairs; the board build stamp stays in the daemon version, the host identity is reported separately.
- The operator performs the swap inside a maintenance slot; the replacement guard stays; the system reopens only with the paired identity and one verified writer.
- Disposable acceptance covers late starts by old clients and all four old/new wrapper and daemon pairings.
- Rollback opens mutable state only after proving the old reader accepts every retained schema, and never erases later records by restoring an old snapshot.

## 3. Phase SH2+: platform canons

Each item below is an **activation gate**: it is designed and tested before it is switched on, and SH1's unchanged tests are not evidence for it.

### 3.1 Service account, state and capabilities

- The host moves to its own service account `swarma-session-host` (home `/opt/swarma/services/swarma-session-host`) only after the direct file APIs (notice inbox writes, context-state reads by the board) are replaced by authenticated calls, and only together with agent-account session runners (§3.2). Running harnesses as the service account is not a boundary.
- The socket directory `/opt/swarma/run/session-host/` has published permissions that let authorised principals connect while the data stays private. Connections follow broker §4.1: kernel peer UID, server identity check, close-on-exec, one role per connection; managed principals by `(uid, generation)`.
- Capabilities: issued by the host to a logical session and its execution, with audience, actions, expiry and revocation; parent and child relationships are authoritative host records; replacement rotates execution-scoped authority; controller epochs are reconciled before effects. The legacy global token is refused after activation. Trusted audit and transcript evidence lives at the service boundary, not in agent-writable files.

### 3.2 Holders under agent accounts and the process tree

Before session runners are activated, the ownership table is frozen:

| Concern | Owner |
|---|---|
| the PTY master or app-server transport | the session runner `swarma-session-runner`, one process per session, started by `swarma-user-launcher` under the agent account |
| the credential lease client (renewals for Codex app-server) | the same agent-side session runner, for the whole execution |
| durable metadata, logs, receipts | the host service |
| admission, allocation and execution records, cleanup | the dispatcher |
| stop | the dispatcher, through the user manager's `exec.stop` (no stop through sudo) |

- The session runner talks to the host over an authenticated channel compatible with the launcher's descriptor rules (plan on descriptor 3, nothing else inherited); a replacement host re-authenticates and reconnects.
- Dispatcher recovery distinguishes an absent controller, a live authenticated session runner, an absent execution and unknown evidence; only confirmed absence permits replacement or cleanup. Controller-only re-adoption keeps the execution id; a new exec gets a new one.
- Start, readiness and exit receipts are acknowledged and replayable; capacity release follows confirmed absence.

### 3.3 Backend activation with the dispatcher and the helper

Enabled only with jointly pinned host, dispatcher and helper versions:

- the ticket schema and signature, issuer and host binding, canonical plan digest, executor, expiry, active generation and live claim; redemption journaled atomically with the launch intent and receipt; a lost reply resolved by the same effect id and receipt, never a second start; a defined cancel-versus-redemption interlock;
- a helper `host` role that may redeem tickets for generations the dispatcher owns;
- identities kept distinct: account generation, execution id, ticket epoch, and the backend's controller epoch;
- the interactive and resume subset negotiated explicitly (dispatcher v0 refuses them);
- negative cases: wrong principal, replay, altered plan or executor, expired or revoked claim, crash on each acknowledgement.

### 3.4 Effects, fences and admission

- **Effect matrix.** Every effect — enqueue, deliver, steer, interrupt, goal and control calls, attach, start, wake, retry, probe, session runner re-adoption — has its fence and epoch rule and idempotent `op_id` behaviour. A closed fence may queue but not deliver; only the operation holding the fence may restart; an explicit stop stays stopped; after a controller failure nothing writes to session runners before reconciliation.
- **One node admission authority** relates the dispatcher's capacity and execution claims, the host's fences, ticket redemption and the permits held through start or verified re-adoption. Every starter is inventoried (board runners, manual and alias launchers, restart and recovery consumers, probes, retries, fallback legs, mixed-version clients); an unsupported participant blocks shared mutation. Unknown permits stay owned; unreadable state is closed; ordinary restart closure is separate from an operator's update lock, which only the operator clears.

### 3.5 Park, restart, emergency stop, board completion

- **Park** is a durable operation: the board acknowledges a committed goal revision that carries the handoff reference (compare-and-swap); the note attempt is bounded, with an explicitly marked snapshot fallback; a timeout never proves drain.
- **Restart** follows the architecture's transaction: drain external effects and child tools, confirm process and native-lock absence, check binding, posture, authentication, grants and runtime, initialise under a closed fence, reconcile unknown delivery, then reopen.
- **Recovery matrix:** provider × owner kind (primary, owner-run) × source record (hosted, legacy) × recovery mode, each row with its fresh plan and credential supply, authority, home and native identity and readiness checks; unsupported rows stay visibly parked or refused; legacy ambient environments are never replayed into the new account.
- **Board completion** is a separate barrier: an execution's exit never satisfies the board's `wait`; exit receipts are durable and replayable.
- **Emergency stop:** an operator command with defined scope and authority, a board snapshot first, fence and absence receipts, and explicit recovery rather than automatic restart.

### 3.6 Existing probes

SH1 preserves the existing adapter paths as they are: Muse PTY's `--version` preflight, the MSP initialize handshake and its identity check, and the last-known-good fallback start. They do not meet the platform's isolation rule. Before platform activation each startup, resume, probe and fallback path gets an adopted qualification policy: approved artifact identity before exec, isolated state or central qualification, a registered attempt, exact result binding and confirmed teardown; each fallback leg gets its own admitted execution. Without that evidence the new platform route refuses.

### 3.7 Wake and liveness

No hosted agent re-arms its own wake-up; there is no `wait` for the model to forget.

| Responsibility | Owner |
|---|---|
| receive mail for an agent identity and hold it durably | the bridge (carrier provider and Mailbox); it enqueues into the host's per-session inbox and counts delivery only after the host's durable write |
| decide when to deliver (the doorbell) | the host, the only writer of a session's input; it delivers at a turn boundary |
| report the turn boundary, idleness and heartbeats | the session runner (§3.2), through the adapter's readiness grammar: Claude PTY or stream output, Codex app-server turn events; an unknown readiness parks rather than guesses |
| detect a wedged session | the host: pending inbox items, and the session neither idle nor progressing past its deadline |
| recover | the host's restart transaction (§3.5) under a new controller epoch, then redelivery, idempotent by sender and message id; the dispatcher only for execution absence and cleanup (§3.2) |
| never woken | an explicitly stopped or parked session, a closed fence (it queues, it does not deliver), a session that needs re-authentication (parked) |

A wake is an effect in the §3.4 matrix: fenced, epoch-checked, with an idempotent `op_id`. The binding from an agent identity to a session names the session, the execution and the epoch, so a wake after a restart never reaches a stale session. Claude Code and Codex are covered; Muse after its qualification (§3.6).

Until SH2+ is active, interim rails live outside this module: a per-identity watcher that holds the wait itself and injects one fixed line only into an idle session with an empty input line and never into one the operator stopped; and, for sessions in today's daemon, its notice path, which delivers only while a client is attached (§1.1).

### 3.8 Goals across accounts

The board, the host and the agent run under three different accounts:
- the board runs under the orchestrator's account or its service account;
- the host runs under `swarma-session-host`;
- the session runner and the harness run under the agent's account.

A goal crosses the boundaries like this:
1. The orchestrator sets or updates a goal on the board, as today.
2. The board's goal binding calls the host's socket: apply revision N to session S. The host identifies the caller by its kernel peer credentials (UID and active generation). It requires a capability for S that it issued to that principal when the session was created at its request; the capability carries audience, actions (apply and roll back goals, deliver, stop), expiry and revocation. The host also requires S's fence to be open.
3. The host records the revision durably. At the next turn boundary it hands the revision to the session runner of the current epoch, which writes it into the harness input (§3.2, §3.7) and returns the harness's acknowledgement.
4. The host answers the board: applied, deferred, or rolled back with the joined error, as in today's goal seam.

An agent cannot set its own goal: its account has no goal capability on the host socket, and the session runner accepts input only from the host. A session runner proves it belongs to S's execution through the launcher ticket it was started with, and the host talks only to the session runner of the current epoch. Revoking the orchestrator's grant or the session capability stops goal operations at once; stopping the execution stays with the dispatcher (`exec.stop`).

In SH2+ the module also ships the host service binary, and the board becomes its client over this socket instead of compiling the session manager in.

## 4. Estimates

Stated velocity (2026-10-09): a clean accepted change takes 3–4 hours from producer start to landing; one rework or base movement makes it 10–12 hours; two runs at a time on the shared host. These are serial sums, not commitments; the landing time is already inside the rate.

| Work | All clean | One rework each |
|---|---:|---:|
| SH1 module landings F, O, G, X, Y, Z | 18–24 landing-hours | 60–72 landing-hours |
| C1 consumer cutover | plus a maintenance slot after 19 October | |
| SH2+ (15–25 changes across host, dispatcher and helper) | 45–100 landing-hours | 150–300 landing-hours |

With continuous agent coverage, SH1 fits in 3–4 calendar days clean and about 5–7 with rework, if F passes; SH2+ in about 1.5–2.5 weeks clean. The critical path is the jointly versioned host, dispatcher and helper activation (§3.2–3.3) and the consumer acceptance, not the copy. CI times are measured at F and added only where they are not already inside the rate.

## 5. Dispositions

| Item | Disposition |
|---|---|
| frozen-plan gap R1 (resize) | §2.3 |
| D1 (restart closure) | §3.4 |
| D2 (probe isolation) | §3.6 |
| cutover through the private fork | superseded: the board consumes a tagged public module through C1 |
| operator-only deployment and update-lock clearance | §2.6, §3.4 |
| CI capacity | GitHub-hosted runners for the module; the board's own CI for the consumer lane |
| calendar gate | 19 October (owner decision 2026-10-09) |
| r3 review F1–F12 | §1.1 and §2.1 (F1), §2.3 (F1, F2), §2.4 (F3), §2.2 and §2.5 (F4), §2.6 (F5), §3.3 (F6), §3.2 (F7), §3.1 (F8), §3.4 (F9), §3.5 (F10), §3.6 (F11), §4 (F12) |

## 6. Out of scope

Terminal backends other than the per-session session runner (tmux), the `ax` backend, the three-operating-system demo, GUI and mobile distribution.

## 7. Open questions

1. Which board key replaces `BoardKey` once one host serves every board on a node (must be settled before multi-board admission and capabilities in SH2+).

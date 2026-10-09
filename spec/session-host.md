# curator-session-host: specification

- **Status:** draft r3 (2026-10-10). Nothing has been moved yet.
- **Normative language:** MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.
- **Lineage:** this revision continues the earlier session-host specification and the as-is extraction plan (revision 2, frozen as the reference alternative on 2026-10-05). The owner chose the as-is path on 2026-10-09: extract one to one first, then bring the host to the platform's rules. This revision closes the plan's known gaps, adds the Muse hosts that have since landed, and drops the terminal-backend and `ax` milestones.
- **Companions:** [curator-dispatcher](https://github.com/relux-works/curator-dispatcher) (admission, allocations, executions; the session-host backend contract), [curator-host-helper](https://github.com/relux-works/curator-host-helper) (agent accounts, the launcher, `exec.stop`), [curator-credential-broker](https://github.com/relux-works/curator-credential-broker), [agent-session-bridge](https://github.com/relux-works/agent-session-bridge) (input producers and output consumers), the platform architecture §5.1 (launch contract), §5.3 (fences and restarts), §7.5 (isolation).

## 1. What the host is

The session host keeps agent sessions alive in the background. It starts a harness in a given directory with a given goal, lets a person attach and detach, delivers messages into a session (the doorbell), and reports events. Today this is `tb-sessiond`, the session manager inside task-board (`tools/board-cli/internal/sessionmanager` in relux-works/skill-project-management).

### 1.1 Today, in short *(fact)*

- One control plane per board, keyed by `BoardKey`; state in the user state directory; self-restoring start-up.
- Root (primary) sessions started by `task-board claude|codex`; owner-run sessions for goal-bound spawns.
- Goals delivered as each harness's native goal; continuation owned by the harness's evaluator.
- In-session notices: a durable inbox per session, delivered only while a client is attached.
- Provider hosts behind the `ProviderHost` / `ProviderHostFactory` contract: Claude in a PTY the daemon owns, Codex through a private app-server per session, and Muse (a PTY host and an MSP provider host, landed since the plan's revision 2).
- A Claude session dies when the daemon is replaced, because the daemon holds its PTY master.
- Access control: one bearer token in a private file; any process of the operator's user can use it.

## 2. Phase SH1: extraction as it is

The first step moves code, not behaviour.

### 2.1 What moves

| Group | Contents |
|---|---|
| provider hosts | the `ProviderHost` / factory contract and optional capabilities; Claude PTY host, output, transcripts, assay and context machinery; Codex host, context, auto-continue and native goals; the Muse PTY host and the MSP provider host with their goals and dispatch |
| terminal | the terminal relay, attachments, **including resize** |
| notices | the durable inbox and the delivery pump (the board still produces notices) |
| records | key, layout, record store, operator log, liveness, resume, repair, quarantine |
| control | client, control protocol, daemon, manager, takeover guards, detached launch, instance lock, shutdown |
| context machinery | builder, context, materialization, consumer observation, enforcement key, provider access and proxy, split along §2.3 |
| daemon skeleton | instance lock, control socket, self-restoring start-up, health reporting, parameterised by a key (today the board key) |

The production and test files of the package and its sub-packages move as a snapshot of a pinned source commit, without the source repository's history; the commit is recorded with the move, and the NOTICE keeps the original attribution.

### 2.2 What stays in task-board

The board and its goal ledger, routing, the spawn runtime, run manifests, acceptance and handoff evidence, repository operations, and the `task-board claude|codex|session|spawn` commands, which keep their names and behaviour and reach the module through the seam of §2.3.

### 2.3 The seam

In-process Go interfaces exported by the module; the board supplies adapters:

- **Sessions:** attach, stop, list, show, open terminal (with initial geometry), **resize terminal**, begin and end owner activation, apply and roll back a goal revision, run an owner turn.
- **Goal ledger and binding:** the board's adapter keeps its write transaction (get, reconcile with a binding, confirm native applied); the host keeps per-owner serialisation; activation leases keep their crash expiry and idempotent release; promotion failure invokes rollback with joined errors visible.
- **Context services:** prepare, complete, fail and list materialisations; reclaim and sweep retention; economics, completeness and builder finalisation. The host owns provider forks, writer leases and local evidence; the board's adapter owns authorisation, selection, canonical publication and board-wide lifecycle.
- **Authority interface** (mandatory; no optional-policy bypass): context, coverage, element, run evidence, retire, publish, recover, retention reports and reclamation confirmation, and the materialisation read snapshot.
- Closed data types preserve JSON shapes, error precedence, rendering and nil semantics; status, detach, takeover, doctor, notice and client-option calls stay compatibility services until mapped.

Provider success in SH1 means exactly what it means today: Claude acknowledges the exact rendered revision; Codex keeps its text-directive delivery; Muse keeps the behaviour its accepted adapter defines. No new capability is invented in SH1.

### 2.4 The gate

- **F (feasibility, first).** Prove the final package graph: the unchanged task-board tests run against the module through a disposable pinned board checkout with the production sources projected at their original path; only declared deterministic import and type adapters are allowed; no test edits, no golden regeneration, no extra skips; every test and fixture is hashed; deliberate mutations of the moved code fail the unchanged tests. If F fails, SH1 stops at the mechanical compatibility milestone and the board keeps its dependencies.
- **Done when:** task-board's session, goal and notice suites pass unchanged against the module; the goldens of every harness are byte-identical; the operator commands behave as before; the module's own CI (vet, race, Unix conformance) is green; any behaviour difference is either removed or listed and accepted by the owner.

### 2.5 Landings

| Landing | Content |
|---|---|
| F | feasibility proof, baseline, module CI |
| O | the snapshot copy and its provenance review (leak scan before the first public push) |
| G | goal and configuration converters |
| X, Y | context boundaries (X: materialisation and authority reads; Y: retention, economics, finalisation) |
| Z | dependency exit and release packaging (immutable module tags, no local replacements) |
| C1 | board delegation through the seam, paired release manifest, cutover |

### 2.6 Freeze and forward-porting

- Before O, the package is frozen in task-board for the copy window, scheduled with the board's arbiter after the Muse work currently landing in the package (expected around 13–14 October). Until then the move is rebased over upstream changes.
- After O, the old source is frozen: every change in flight gets an accepted commit, a destination and one named forward-port integrator in a freeze manifest. No second runtime is authored in parallel.

### 2.7 Cutover

- Not before the board's conference window ends (19 October).
- A paired release manifest names the board build, the module tag, the wire, goal and storage schema versions, and the permitted reader and writer pairs. The board build stamp stays in the daemon version during SH1; the host identity is reported separately.
- The initial cutover is the operator's act inside a maintenance slot: stop the daemon outside any hosted session, prove absence, snapshot state, install the paired binaries, start with one writer; the replacement guard stays. Rollback opens mutable state only after proving the old reader accepts every retained schema; otherwise it stays stopped with records intact.

## 3. Phase SH2+: platform canons

After SH1, behaviour changes land one gate at a time.

### 3.1 Access control (first)

- The host runs under its own service account `swarma-host` (home `/opt/swarma/services/swarma-host`).
- Its socket lives in a private directory under `/opt/swarma/run/host/`; every connection is identified by the kernel-reported peer UID, as in broker §4.1.
- Each session gets a scoped, revocable capability: a session acts on itself and its own children; operator calls carry the operator's own authority; a supervision matrix decides who may stop, park or attach to whom; every call is audited.

### 3.2 Holders under agent accounts

- A hosted session runs under its agent's OS account. The host never runs `sudo -u`; it asks for a start through the host launcher with a **launch ticket** issued by the dispatcher for that generation and execution (dispatcher §12). The launcher accepts a host launch only with a valid, unused ticket.
- The executor `curator-executor` runs first under the agent account, leases the credential and execs the harness; the host keeps the terminal through a per-session holder process so that a host restart no longer kills Claude sessions.

### 3.3 Admission, fences and epochs

- The dispatcher admits and records executions; the host owns holding, readiness, exit receipts and fences. Calls between them carry `(logical_session_id, execution_id, controller_epoch, op_id)`.
- A durable per-session fence (`open`, `closed`, `update_lock`) is checked atomically with every start, wake, retry and probe; a supervisor start increments the controller epoch, and holders accept commands only from the current epoch.
- An atomic node-wide admission (permits serialised with admission closure) protects updates: closing admission fences new permits and accounts for outstanding ones; an unknown permit keeps ownership; only the operator clears an update lock.
- Ordinary controlled restart has its own durable closure, acquired atomically with the permits, retained across replacement and released after authenticated start-up reconciliation (gap D1).

### 3.4 Delivery, park, emergency stop

- Doorbell modes: `next_turn` by default, `steer` into a running turn where the harness can take it, `interrupt` only for halt, park and emergency stop. Delivery no longer waits for an attached client; the event stream makes detached turns visible.
- Park: the session writes a handoff note bound to the board goal revision, and the next start resumes from it.
- Emergency stop: a separate, unmistakable operator command with a board snapshot first.

### 3.5 Probes and harness updates

Runtime capability and resume probes stay disabled until they run with the same isolation as the adapter probe (a sandbox or central qualification), with authenticity and known-bad checks before execution, a registered attempt, an unchanged original state on failure and proven teardown (gap D2). There is no unsandboxed local execution of a fresh harness build.

## 4. Estimates

Measured velocity in this stream (2026-10-09): a small accepted change takes 3–4 hours from producer start to landing; with one rework and base movement 10–12 hours; a large change can take four or five revisions; two runs at a time on the shared host.

| Step | Work | Elapsed |
|---|---|---|
| SH1 in the module's own CI | F (about half a day), O with provenance review, G, X, Y, Z | 3–4 days without large reworks, 5–7 days realistically |
| SH1 cutover into task-board | C1, paired manifest, maintenance slot | 1–2 days, after 19 October |
| SH2+ | access control, holders under agent accounts with launch tickets, fences and epochs, admission, park and stop | about 15–25 changes; 1.5–2.5 weeks |

## 5. Known gaps from the frozen plan, and their disposition

| Gap | Disposition |
|---|---|
| R1: the neutral terminal seam dropped resize | resize is part of the Sessions seam (§2.3); the existing resize test binds to it |
| D1: controlled restart needs its own durable closure | §3.3 |
| D2: probe isolation | probes stay disabled until isolated (§3.5) |
| cutover through the private fork | superseded: the module is public; the board consumes a tagged module through C1 |
| deployment is the operator's act | §2.7 |
| CI capacity | the module runs on GitHub-hosted runners; re-measure CI waits at F |
| update-lock clearance is the operator's alone | §3.3 |
| calendar gate | the board's window ends 16 October; this plan keeps 19 October as its margin (owner decision 2026-10-09) |
| budget gate | the estimates assume codex capacity; a budget stop extends them |

## 6. Out of scope

Terminal backends other than the per-session PTY holder (tmux), the `ax` backend, the three-operating-system demo, GUI and mobile distribution.

## 7. Open questions

1. Whether the module also absorbs `tb-sessiond`'s binary and command line in SH1, or task-board keeps building the daemon until C1.
2. Which board key replaces `BoardKey` once one host serves every board on a node.

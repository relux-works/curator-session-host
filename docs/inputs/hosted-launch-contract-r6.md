# Hosted-launch contract r6 input — task-board receiver superseded

The task-board receiver targeted by this draft is **superseded by the owner's decision of 2026-10-12**: hosted launch folds into the swarma dispatcher path, `start {plan, plan_digest, executor}`, with the session-host executor. This file retains useful design input only. It publishes no swarma payload, receiver, admission-record or adapter schema and activates no hosted route.

- **Revision:** input harvest (2026-10-13), limited to the seam payload, admission records and imported-Claude adapter notes. The current [session-host specification](../../spec/session-host.md), including R-START/R-SESSION and the later owner decisions, takes precedence.
- **Provenance:** [agent-session-host PR #10](https://github.com/relux-works/agent-session-host/pull/10), “Add the hosted launch contract (r6) and a Russian one-page overview”; head commit `d7a732f3521952e637357ae09decfa8d47465919`. Source: [`spec/hosted-launch-contract.md` at that head](https://github.com/relux-works/agent-session-host/blob/d7a732f3521952e637357ae09decfa8d47465919/spec/hosted-launch-contract.md), principally §§3, 3.1, 4, 5 and 5.1. The PR diff was read without modifying the source PR. The Russian overview is not retained.

## Seam payload input (source §3)

The useful seam is a concrete process projection composed once by the provider module. The host consumes it without reconstructing policy, MCP settings, home or environment. The historical private payload's names below describe that input, not fields added to the swarma `start` envelope:

| Source group | Retained input |
|---|---|
| `schema`, `schema_version`, `content_digest`, `producer` | Closed, versioned, locally pinned shapes; producer module version/commit; independent integrity recomputation. The payload digest is transient integrity, not authentication, durable provenance or resume identity. |
| `environment`, `process` | Provider plus exact binary, ordered argv, full ordered environment, cwd, stdin and imported execution guard. Empty environment means empty, never daemon inheritance. |
| `owned_literals`, `env_names`, `managed_home`, `fragment` | Module-owned values, required caller-environment names and resolved home/profile provenance. Preserve absent versus present-empty values; reject duplicate names, malformed entries and NULs. Never substitute daemon environment. |
| `policy` | Resolved policy and its source, with module-validated effective native policy. The old draft's permission defaults are not adopted here. R-EXEC governs the platform execution profile. |
| `session_name`, `remote_control` | Host and native names are distinct. Remote-control positions refer to the final primary argv, excluding the binary; settings-origin intent may have no argv positions. Support remains subject to module qualification. |
| `resume`, `restart` | Module-normalized intent and a validated restart template with a declared identity insertion. The receiver does not parse provider selectors or infer a conversation identity. |
| `host_bundle`, `network` | Historical extension points only. No bundle or network schema is adopted; R-NET-TOOL and network-profiles N15 govern current platform boundaries. |

The source distinguishes the frozen private payload from later proposed finalization/binding extensions. Neither is a swarma wire format. Its schema versions, bounds, reservation protocol, terminal descriptors and diagnostics registry are not copied into the platform contract. FD3 on the current launch path carries the plan only; the runner channel remains unpublished in spec §7.2.

Useful validation requirements are strict lexical/structural bounds, nested schema dispatch, rejection of duplicate keys and invalid text, and independent semantic/source checks before resource allocation. A full environment carries caller credentials only through the admitted transient credential path. Values, request bodies and secret-dependent digests do not enter diagnostics, logs or durable snapshots. Secret-free metadata must be validated against admitted sources, rather than accepted merely because its shape is valid.

The source's finalization idea is useful: the module inserts only explicitly declared identity slots and updates derived final-argv indices without changing intent or unrelated metadata. Auxiliary projections retain their own module-declared argv; they do not acquire the primary conversation UUID. The old task-board reservation service is superseded, and its bindings require a joint swarma publication rather than direct reuse.

## Admission-record input (source §§3.1, 4 and 5.1)

Admission precedes durable records, leases, terminals, auxiliary probes and provider I/O. The receiver and the executing side validate independently; a caller-side check cannot replace the executing side's check. In the current platform, the dispatcher owns admission/allocation/execution records, the host owns session state, and the board is optional. The source's board-only scope and board-keyed receiver are not retained.

The record inventory is useful for deciding what can survive restart, with every nested reference closed and versioned:

| Historical record group | Useful distinction for current design |
|---|---|
| Record schema/version; session/provider/native identity; owner, board fingerprint and canonical project; lifecycle/timestamps; host/native names and remote-control intent | Keep logical session identity, provider continuity, caller authority and optional board binding explicit. A board fingerprint is relevant only when a board is bound. |
| Admitted module/guard-schema versions; fragment; resolved policy; safe restart-template reference; environment/literal names; non-secret home selectors; sanitized network record | Persist validated credential-free references, rather than a full launch DTO or arbitrary provider blob. Restart references omit original selectors and one-shot stdin. |
| Record revision; execution counter; current role executions; completion; provider host metadata | Keep execution attempts and completion separate from logical sessions, attachments and daemon instances. Stale completion must not authorize actions on a replacement. |
| Credential-free provenance digest and keyed literal commitments | Separate stable provenance from transient secret-bearing process material. Bare hashes of credentials/values are not durable substitutes. Missing commitment keys or renewal material cannot be interpreted as record absence. |

The source's `ProviderHostRecord`/`Opaque` discussion calls for closed provider metadata: identity, process role, daemon incarnation, kernel birth evidence, safe restart and credential-free reconnect references. Arbitrary JSON/string blobs and credential-bearing URLs are excluded. Full plans, environment/literal values, stdin, raw guard/bundle data, wire digests and control credentials do not enter records, snapshots or logs. Only a closed non-secret home-selector map was proposed to retain selector values; its swarma shape remains unpublished.

The historical `launch_generation` is a per-record execution counter, **not** the platform's random account generation or host controller epoch. No field or counter scheme is adopted here. The current specification keeps account generation, execution identity and controller epoch distinct (§3.3).

Useful completion input is generation-bound authenticated process completion persisted before it is advertised. Attachment detach or transport loss does not invent a child exit status; terminal EOF alone proves neither success nor permanent absence of completion. A failed or partial read is unknown, not a missing record. Persisted completion or conclusive absence is required before releasing an execution's ownership. Current account-wide stop and absence proof remain with the user manager (F-13/R-FATE), not an imported adapter's PID-based stop logic.

Ordinary reuse/resume checks stable provider, canonical project, profile pin, native identity and safe restart provenance. Changed provenance needs explicit migration rather than silent renewal. Unknown liveness or a live but unattachable process retains the slot; it does not trigger a duplicate launch. Legacy records must not silently become hosted records. The source's missing-volatile-material distinction is useful, but its proposed parked-state label is not introduced as a platform enum.

## Imported-Claude adapter input (source §§3, 5 and 5.1)

The source reports module APIs for selector elevation, restart-template export/validation, seal import/export and plan finalization. These are source-reported APIs at the PR head, not evidence of present support or an activation claim:

- `agentic.ElevateResumeIntent` moves native selectors into typed intent using provider grammar. Conflicting, ambiguous or picker identities cannot be guessed by the receiver.
- `claude.System.ExportRestartTemplate` and `(*claude.System).ValidateRestartTransformation` validate the declared identity insertion without recomposition. Restart refreshes permitted volatile material and omits one-shot stdin.
- `Plan.ExportSeal` / `agentic.ImportSeal` distinguish local plans from transported guards. Importing a seal does not install `Plan.VerifyBeforeExec` on a local plan.
- `agentic.FinalizePlan` was reported to reject typed bindings at the source revision; admissible imported-process finalization remained a gate. The source proposed `agentic.NewImportedProcess(seal, process)` and `ImportedProcess.VerifyBeforeExec` to bind the exact process and reject differences immediately before each start. Delivery/tagging must be established separately.

Claude's historical marker is unsealed. It proves neither producer identity nor artifact integrity. A nil verifier being a no-op is not verification evidence. A transported marker needs a closed import, exact process/restart validation and the shared native binary/MCP/prompt checks through a public execution-free module API. The daemon must not import launcher internals. Managed-argument and goal-readiness checks remain separate. Do not relax schemas, downgrade guards or assume that a new payload version supplies missing bound-import support.

Every primary, auxiliary, preflight or assay uses its concrete module projection, including its final environment/home/settings. Mandatory version/goal probes and any required context assay need module-owned projections and verification; the host does not author their argv or borrow legacy probes. Verification must sit immediately before each admitted start, not solely at the receiving seam. In swarma that start belongs to the admitted executor/runner under R-EXEC, never a self-started harness in an agent account.

The source distinguishes primary conversation identity from auxiliary execution identity. A primary identity is bound before spawn, not inferred from the newest transcript. Auxiliary argv never receives the primary conversation UUID. Historical auxiliary cancellation required the retained original process handle plus fresh kernel birth/role/execution evidence; handle loss or incomplete process/group lookup remains unknown, and auxiliary completion cannot prove primary death. These are adapter qualification inputs, not permission for the session host to bypass the platform's journaled account-wide `exec.stop` and confirmed-absence path.

The source's daemon-owned Claude PTY cannot be recovered merely from a live PID. That is useful SH1 extraction input, not a survival promise for SH2+. The current runner owns the PTY, and R-FATE requires published process-class fate and per-platform evidence. Native-resume compatibility and failure behavior follow D-RESUME; the source's automatic fallback/version policy is not retained.

## Still unpublished

Joint publication would need the swarma plan projection and adapter-import schemas, credential-free record/reference formats and migration rules, allowed identity finalization and verification evidence. Spec §7.2 owns lifecycle/runner-channel publication; §7.13 owns workspace transfer; §7.14 owns task-resource and web-tool formats/qualification; §7.15 owns execution profiles and standing surfaces. No old refusal code, reservation token, numeric bound or coordination protocol is adopted by this harvest. Web remains default-off with explicit per-identity trusted-circle opt-ins only; read-only task resources and all host boundaries follow R-NET-TOOL/R-UNTRUSTED.

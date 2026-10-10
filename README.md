# swarma-session-host

The session host keeps agent sessions running in the background: it starts a harness (Claude Code, Codex, Muse) in a given directory, lets a person attach and detach, delivers messages into a running session, and reports what happens. A goal is optional. The module must work without goals, a board, an orchestrator or their toolchain; task-board is an optional client of its generic goal interface. Today this job is done by the session manager inside task-board (`tb-sessiond`). This repository is where it becomes its own module.

**Status: design.** No code has been moved yet. The plan is in [`spec/session-host.md`](spec/session-host.md):

1. **Extraction as it is (SH1).** The provider hosts (Claude PTY, Codex app-server, Muse), the goal and notice delivery, transcripts, records and the daemon skeleton move here with no change in behaviour; task-board reaches them through a thin seam; task-board's existing session, goal and notice suites are the gate.
2. **Platform canons (SH2+).** Then the host is brought to the platform's rules: the `swarma-session-host` service owns durable state, fences and input admission; one `swarma-session-runner` per session owns the PTY master or private Codex app-server transport under `worker-<label>`, started by `swarma-user-launcher` through `swarma-dispatcher`. The dispatcher stops executions through `swarma-user-manager`'s `exec.stop`. Peer credentials, per-session capabilities, parking and controlled restart are activation gates.

For now, only the board/orchestrator for board goals, or the launching caller at start through the launcher, authors a session goal. The agent account has no goal capability on the host socket. This keeps authorship with the initiating caller while the bounds of an agent namespace await design. A separate SH2+ item will choose between agent self-goals in a separate namespace and retaining caller/board authorship only; this does not block a goal-free host.

Same-machine native resume trusts non-breaking harness semver changes: the same major, and for 0.x the same minor, with the continuity preconditions in spec §3.5. A different major or 0.x minor is refused. A harness resume failure is reported as `resume-failed` to the caller, which decides the next action; there is no automatic fallback or portable seed on this path.

Related modules: [swarma-dispatcher](https://github.com/relux-works/swarma-dispatcher), [swarma-user-manager](https://github.com/relux-works/swarma-user-manager), [swarma-credential-broker](https://github.com/relux-works/swarma-credential-broker), [curator-agent-launcher](https://github.com/relux-works/curator-agent-launcher) (`curator-run`), [agent-session-bridge](https://github.com/relux-works/agent-session-bridge).

The generic SH1 gate tools and invented fixture are documented in
[`docs/sh1-gate.md`](docs/sh1-gate.md). Public CI exercises the tooling on
GitHub-hosted macOS and Ubuntu. Lane 1 runs in public module CI with a private,
pinned board checkout; the owner accepted board code excerpts in public CI logs
on 2026-10-10. The current public lane 2 prototype builds and vets the consumer
through a file GOPROXY; the release gate requires the full private board CI
against an immutable public module tag, without a candidate replacement. Lane 3
tooling checks are distinct from future extracted-module conformance. The gate
records failures and never hides skips; see the evidence and declared constraints
in the gate document. No test or build runs on the development host.

Through C1 the board will consume the tagged public serving library while
retaining the `tb-sessiond` executable and composition root. This module's
private-fork cutover plan is superseded; the existing tooling does not establish
extraction or runtime migration. Operator maintenance, quiescence and paired
release identities remain required. New ownership, recovery, state and gate
diagrams are deferred to R-SH2; this repository currently has none.

## Provenance

Code will move here as a snapshot of task-board's `tools/board-cli/internal/sessionmanager` at a pinned commit of relux-works/skill-project-management (Apache-2.0), without that repository's history. Each move will record the source commit. The NOTICE will keep the original attribution.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE) for licensing and attribution.

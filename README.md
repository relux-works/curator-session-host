# swarma-session-host

The session host keeps agent sessions running in the background: it starts a harness (Claude Code, Codex, Muse) in a given directory with a given goal, lets a person attach and detach, delivers messages into a running session, and reports what happens. Today this job is done by the session manager inside task-board (`tb-sessiond`). This repository is where it becomes its own module.

**Status: design.** No code has been moved yet. The plan is in [`spec/session-host.md`](spec/session-host.md):

1. **Extraction as it is (SH1).** The provider hosts (Claude PTY, Codex app-server, Muse), the goal and notice delivery, transcripts, records and the daemon skeleton move here with no change in behaviour; task-board reaches them through a thin seam; task-board's existing session, goal and notice suites are the gate.
2. **Platform canons.** Then the host is brought to the platform's rules: its own service account, a private socket with peer credentials and per-session capabilities, holders started under per-agent OS accounts through the host launcher and the dispatcher, durable fences and controller epochs, parking and controlled restart.

Related modules: [curator-dispatcher](https://github.com/relux-works/curator-dispatcher), [curator-host-helper](https://github.com/relux-works/curator-host-helper), [curator-credential-broker](https://github.com/relux-works/curator-credential-broker), [agent-session-bridge](https://github.com/relux-works/agent-session-bridge).

The generic SH1 gate tools and invented fixture are documented in
[`docs/sh1-gate.md`](docs/sh1-gate.md). Public CI exercises the tooling on
GitHub-hosted macOS and Ubuntu; lanes 1 and 2 also run here against a pinned checkout of the board. The gate records failures and never hides skips; see the evidence and declared constraints in the gate document.

## Provenance

Code moved here is a snapshot of task-board's `tools/board-cli/internal/sessionmanager` at a pinned commit of relux-works/skill-project-management (Apache-2.0), without that repository's history. Each move records the source commit. The NOTICE keeps the original attribution.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE). Authors: Ivan Oparin and Alexey Grigorev.

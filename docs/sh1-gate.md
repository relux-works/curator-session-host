# SH1 gate interface

F part 1 adds generic tooling and an invented fixture. It copies no package
implementation. Green public tooling CI validates the tools; it does not prove
that the private session manager can be extracted, or activate any SH2 capability.
The recursive source snapshot and the actual three-lane proof remain later work.

## Lanes and ownership

The owner decision of 2026-10-10 supersedes the original private workflow plan.
`sh1-lanes.yml` runs on pushes to this public repository's own branches and on
`workflow_dispatch` with `spm_ref` (default `main`). A hosted Ubuntu job resolves
that ref once; all three matrix jobs check out the same exact board commit.
No pull-request event can access the board token through this workflow.

| Lane | Location | Input and gate |
| --- | --- | --- |
| 1: projection | GitHub-hosted macOS and Ubuntu | Recursive candidate projected at the original import path; unchanged root and `musepty` tests, uncached and unmasked; inventories, guard ratios and named-test mutations. |
| 2: consumer | GitHub-hosted Ubuntu | Candidate through the file GOPROXY, no candidate replace; `go build ./...` and `go vet ./...` for `tools/board-cli`. |
| 3: tooling | GitHub-hosted macOS and Ubuntu | Generic tool vet, race tests, build and fixture; this does not establish extracted-module conformance. |

Every job grants only `contents: read`. Actions are pinned by commit SHA;
checkout credentials are not persisted. `SPM_READ_TOKEN` is used only as the
`token:` argument of a board checkout. The owner permits board code excerpts in
public job logs. Lane artifacts upload only explicitly listed JSON manifests,
inventories, projection/guard/mutation/consumer reports and timing summaries.
They never upload board checkouts, candidate bytes, dependency caches, proxy
archives or raw logs. Existing tooling artifacts contain the invented fixture
only. All tests, builds and vet run on GitHub; none run on the development host.

## Commands

Build the standard-library-only CLI with Go 1.25.5 (the source module's Go
version). Paths below are relative to a disposable workspace. Keep reports outside
input trees. Every subcommand requires every documented flag, returns 0 on success
and 1 on refusal, and writes stable indented JSON with a trailing newline.

```sh
go build -o sh1 ./tools/sh1
./sh1 generate -source source -package internal/session \
  -old example.org/sh1-fixture/internal/session -new example.org/sh1-host \
  -mod candidate-go.mod -out candidate
./sh1 project -candidate candidate -checkout projection \
  -package internal/session -commit "$SOURCE_COMMIT" \
  -adapters adapters.json -report evidence/project.json
go -C projection test -json -count=1 ./... > evidence/tests.jsonl
./sh1 inventory -source projection -events evidence/tests.jsonl \
  -guards guards.json -report evidence/inventory.json
./sh1 mutate -checkout projection -mutations mutations.json \
  -report evidence/mutations.json -- go test -json -count=1 ./...
./sh1 consumer -candidate candidate -checkout consumer \
  -version v0.0.0-20261010000000-0123456789ab \
  -proxy proxy -report evidence/consumer.json
```

`candidate-go.mod` is a reviewed complete module file declaring the new module
and the same Go version as the source. Its dependencies are explicit caller
inputs; the generator does not guess neutral interfaces, copy unrelated internal
packages, or resolve dependency versions. Candidate replace directives are
refused. Source modules may have unrelated replace directives; none is copied
implicitly. Unsupported dependency graphs require a design decision at F.

The CLI accepts lowercase module paths with a dotted domain and ASCII letters,
digits, `._-/`; v0 pseudo-versions are the supported initial proxy interface.
Relative file declarations must be normalized, nonempty, and remain inside their
tree. Symlinks and other nonregular files are refused. Inputs and outputs are
trusted local workspaces, used serially with no concurrent writers. This tool
is not a security boundary for an adversarial filesystem or forged manifests.

### generate

Copies every file recursively, including tests, fixture programs, testdata and
non-Go files. Rewrites only Go AST import literals matching the old path exactly
or its slash-prefixed descendants; comments, string data, and file names stay
unchanged. Reserved root `go.mod` and `sh1-manifest.json` collisions are refused;
the output must be absent or empty. A supplied candidate module file is emitted
as `go.mod`.

`sh1-manifest.json` lists each output file, mode, original SHA-256, transformed
SHA-256, source go.mod digest, and protected-file classification. The supplied go.mod has no original
package hash. The manifest cannot hash itself and is excluded from its own file
list. The generator digest hashes sorted names and SHA-256s of its embedded Go
sources (including tool tests). The transform digest binds that digest, package,
old and new imports, source go.mod digest and supplied module-file digest. Local
absolute paths and timestamps are excluded. Hosted evidence must bind the
manifest hash to authenticated source/candidate identities; a self-authored
manifest alone does not establish provenance.

### project

Requires an exact 40-character source commit and a clean Git checkout. Verifies
candidate bytes, file set, modes and protected classifications against the
manifest, and original package bytes against original hashes before writes.
Maps candidate production imports back to the old path, writes production files
(including non-Go production assets), and leaves `_test.go` and every `testdata/`
file untouched. Before/after hashes of protected files cover the whole checkout.
The report records the source commit, candidate-manifest hash, adapter-table
hash, production transformations, and protected-file hashes.

The only admitted adapter table is `{"version":1,"entries":[]}`. Unknown
versions, omitted entries, and any nonempty adapter table are refused. Future
adapters require a reviewed versioned implementation, not implicit patches.

### inventory

AST discovery covers named top-level Go Test declarations in all `_test.go`
files, including inactive build-tagged files. Package rows list discovered
source tests and observed dynamic subtests, build-tag expressions, opt-in
conditions, execution flags, terminal results and skip reasons. Every skipped
test has a reason; absent or diagnostic-only output is recorded as `unknown`,
including streams without any test output events. Excluded tests
are `not_observed`; this does not infer that they passed or why they were absent.
Go list output is unnecessary for this fixture because source discovery also
sees inactive files. This interface inventories tests, not benchmarks, fuzz
corpus cases, or external packages whose source is outside the checkout.

The input is one uncached `go test -json -count=1` stream, with terminal package
and started-test events. A shared parser validates package/test lifecycles for
both inventory and mutate: one run and exactly one terminal per observed test;
no duplicate or conflicting terminals or unfinished tests at package termination.
Missing, empty, malformed, repeated, partial or
unsupported events refuse. A failed test is recorded as failed; inventory is not
a substitute for checking the original go test exit code. Build failures whose
new event actions are unsupported refuse rather than report absence.

Guard declarations bind package/test pairs to expected relative source targets.
A reviewed guard test emits `SH1_GUARD relative-file` after a successful read.
Only a marker from that named test with a passing terminal event counts.
`guard_ratio` and per-guard ratios measure reached/expected targets; incomplete
coverage writes a report and exits 1. The report does not infer coverage from a
passing guard or from a marker emitted by another test. Bound: markers depend
on reviewed instrumentation; uninstrumented guards remain unproven, and callers
must supply authenticated logs rather than forged event streams.

Opt-in declarations are explicit; the tool does not pretend to infer all
environment-sensitive behavior from syntax. The public fixture reports a
normal opt-in skip, then separately exercises the opt-in and build-tagged tests.
The lane-1 acceptance run must have no skips and no name masks.

### mutate

A version-1 declaration lists unique names, relative files, exact unique anchors,
replacements, and required package/test pairs. Runs the caller's command once as
a baseline and requires exit 0 and every named test passing in JSON evidence.
Each mutation is applied independently, the identical command is run, and the
file is restored before the next mutation or any reported command error.

A kill requires a nonzero command exit, exactly one completed named test
execution with a failure terminal, and its package failure. Baseline admission
requires exactly one completed passing execution of every named test. Compile
failures, malformed logs, another test failing, and
survivors do not count as kills. Reports retain actual exit codes, before/after
hashes, killed/survived/unknown statuses and a measured kill ratio. Any surviving
or unknown mutation exits 1 after writing the report. The fixture proves both
narrowing the admitted identity class and bypassing its gate, plus a by-name
source guard. Later gates declare converter and additional real guard mutations too.
Caller commands must emit Go test JSON on stdout; diagnostics belong on stderr.

### consumer

Validates the candidate manifest, serves sorted deterministic `.mod`, `.info`
and `.zip` bytes from `module/@v/` plus a version list, then edits the consumer's
require with Go's module editor and rewrites matching AST imports throughout
its Go files. It never adds a replace and refuses an existing replacement of
the candidate module. Existing unrelated replacements remain caller-owned.
Conflicting bytes at a published proxy version are refused.

Runs effective module selection, download and module graph commands with a
fresh module cache, GOWORK off, only this file proxy, and the checksum server disabled for the local synthetic
publication. Records command exit codes, the two candidate go.sum lines, the
module graph, proxy file hashes and consumer file hashes. Selection from
`go list -mod=mod -m -json all` must equal the requested pseudo-version; a higher
MVS minimum refuses. Checksums must have exactly the candidate zip and `/go.mod` keys for
that version and match `go mod download -json`. Historical and unrelated
checksum lines are preserved and excluded from the candidate checksum pair.
A v0 pseudo-version labels a disposable candidate; the synthetic timestamp/revision in the example
is not an attestation of a real release. Hosted workflows derive those values
from the pinned candidate commit. Later release gates use immutable tags and
normal release checksum policy.

The caller prepares the consumer at dependency exit: the extracted package is
absent, while consumer composition and command sources remain. The fixture
removes its invented extracted package and tests both the parent and child
through the cmd consumer. Original extracted tests run in lanes 1 and 3. All
consumer dependencies must be available in the local proxy or via existing
reviewed unrelated module resolutions; the tool does not silently fall back to
the network. Run the real consumer build and full test commands with the same
GOPROXY after this preparation step. The public end-to-end test does so.

## What F records

The gate's evidence bundle must include:

- Source commit, source go.mod and recursive source file hashes; candidate commit,
  host tree, candidate module file, candidate manifest and transform digest.
- Generator source digest; workflow, adapter table, helper, test, guard declaration,
  mutation declaration, fixture and golden hashes. Record an explicit empty
  adapter set rather than omit it.
- Exact commands and exit codes, source discovery and runtime inventories,
  skip reasons, active/inactive tags, guard reached/expected ratios and each
  mutation's named-test outcome. No truncated logs or inferred passes.
- Consumer require, candidate module version, go.sum lines, module graph and proxy
  publication hashes; later the immutable release tag and authenticated identity.
- Go version, GOOS, GOARCH, relevant build tags and opt-in environment values,
  runner class, toolchain policy, workflow/run identity and per-job durations.
  Never record secrets, personal paths, private machine names or session links.

Artifacts live with the run and are attached to the Change Request before
handoff. F remains provisional until the actual copy snapshot closes and all
three lanes run on those pinned bytes.

## Hosted lane contract

The helper `tools/sh1/ci/run.py` builds the generic CLI on the runner and derives
its Go toolchain from the board's `tools/board-cli/go.mod`. The reviewed prototype
candidate retains that module's requirements, changes its module path to
`github.com/relux-works/swarma-session-host`, and removes its two unrelated local
board-library replacements. The empty version-1 projection adapter table is
explicit. No tests, fixtures or goldens are patched. The opt-in declaration currently names
the two Claude live smokes and Muse real-provider budget pause. Other
environment-sensitive skips remain recorded with their observed reasons; the
helper does not claim a complete environment-condition inference.

Lane 1 runs exactly:

```sh
go test -json -count=1 -timeout=12m ./internal/sessionmanager ./internal/sessionmanager/musepty
```

The same full command supplies the mutation baseline and every mutant run.
`mutations.json` declares local identity isolation, default-port normalization
(narrowing HTTPS out of the admitted normalization class), and remote board
identity. Each must fail `TestBoardKeysCanonicalizeAndIsolateBoards`. Baseline
failures and compile errors do not count as kills. Full suite failures remain
red even if a named mutant is killed. Inventory discovery is scoped to the two
tested packages at their original module-relative paths, with unchanged test and
fixture bytes copied into a separate inventory tree. Unrelated board CLI tests
are outside this lane's stated bound. Inventories distinguish source declarations
from dynamic subtests; inactive build-tagged tests remain visible. No runtime
skip, failed test, unknown execution or untagged unobserved test is admitted.
Tagged declarations are inventoried without enabling the board's opt-in tagged
mutation suites. This starter list is bounded to identity behavior; converters,
all real guard mutations and released-module conformance remain later gates.

The guard declaration derives the complete `claude*.go` production-file set used
by `TestClaudeContextProductionHasNoFilesystemRemoval`, and also declares
`hosted_takeover_guard.go` for
`TestTheMutantInventoryCoversEveryPredicateOnTheRefusalPath`. Current unchanged
board tests emit no `SH1_GUARD` read markers. A passing test alone therefore
proves no reached target to this checker; the guard gate stays red. A proposed
adapter is reviewed, test-attributed read instrumentation upstream in the board,
with a new pinned source identity; no marker is synthesized from passing tests.
The workflow records the measured reached/expected ratio, including this gap.

Lane 2 mirrors the board's downloaded public dependencies into the file proxy
and publishes the retained `pkg/board` and `pkg/remoteconfig` libraries from the
same board commit at their already required `v0.0.0` versions. These are explicit
prototype dependencies, with publication hashes recorded; they are neither
neutral interfaces nor independently released libraries. The candidate has no
replace. The consumer retains the board's two original unrelated replacements.
The extracted package is removed from the disposable consumer, so tests needing
board-internal access remain covered by lane 1. `consumer` rewrites imports,
selects/downloads the candidate through a fresh cache and the file proxy, records
both candidate go.sum lines and the module graph, and refuses a candidate
replacement or wrong MVS selection. Build and vet then use a separate fresh
cache, the same file proxy, and no network fallback. The pseudo-version is
derived from the public snapshot commit's UTC timestamp and revision; it is not
a released immutable tag. Whole-board CI and the released tag remain Z/C1 work.

Missing qualified providers, Darwin sandbox or PTY restrictions, internal-only
imports and source guards requiring private paths stay visible as failures or
skips with exact test names. No provider stub, name mask, test deletion or
implicit projection adapter is admitted. The full no-skip acceptance gate cannot
turn green until the board owner supplies a hosted-compatible qualified input
contract, including how intentional helper-process skips are handled. The
current run is a development proof, not the Muse-cut freeze evidence.

## Lane 1 and 2 evidence

Hosted run evidence is recorded below after execution. Command summaries record
actual exits and per-command durations; manifests bind recursive source/output
hashes, generator digest, source module digest and transform digest. Inventory
counts, every failed/skipped test, guard targets and named mutation outcomes are
retained as reports. Job durations include checkout, setup and upload and are
measured from the GitHub jobs API, separately from helper durations.

## Public verification and durations

The public matrix runs format, vet, build, and the whole tool test package in
one uncached race-enabled process, including unit/refusal tests and
`TestFixtureEndToEnd`. That test launches the built CLI for every tool and repeats
generation, projection, inventory, mutations and consumer preparation on equal
inputs, requiring byte-identical trees and reports. It executes module and
consumer tests, opt-in/tagged fixture tests, a consumer build and the cmd entry
point. Artifacts include candidate bytes, proxy bytes, reports, inventories,
module graph and sanitized environment identities.

Revision 2 adds `TestMutationRejectsInvalidLifecycleEvidence`, which drives both
CLI entry points with terminal-only baseline/mutant streams, conflicting or
duplicate terminals, duplicate runs and unfinished named/other executions.
`TestLifecycleNarrowingMutant` narrows production enforcement to source guards
and requires that named CLI regression to fail (exit 1, named test and package
failure), recording a 1/1 kill ratio. The ordinary suite remains unmasked; only
the deliberate narrowing experiment uses a named regression mask. Real fixture
kills and compile/unrelated-failure refusals remain in the same suite.

| Run | Ubuntu job duration | macOS job duration | Result |
| --- | --- | --- | --- |
| [38015721398](https://github.com/relux-works/swarma-session-host/actions/runs/38015721398), initial snapshot | 40 s | 59 s | Both green |
| [38015843173](https://github.com/relux-works/swarma-session-host/actions/runs/38015843173), strengthened refusal tests | 31 s | 43 s | Both green |
| [38016057110](https://github.com/relux-works/swarma-session-host/actions/runs/38016057110), implementation and workflow verification | 43 s | 62 s | Both green |
| [38019374197](https://github.com/relux-works/swarma-session-host/actions/runs/38019374197), revision 2 regressions and narrowing proof | 42 s | 48 s | Both green |

Durations are measured from the hosted jobs' started/completed timestamps,
including setup and artifact upload. All recorded snapshots passed every
public matrix step. These measurements concern synthetic tooling only; source/consumer lane durations are recorded in the evidence section above.

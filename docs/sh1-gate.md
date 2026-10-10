# SH1 gate interface

F part 1 adds generic tooling and an invented fixture. It copies no package
implementation. Green public tooling CI validates the tools; it does not prove
that the private session manager can be extracted, or activate any SH2 capability.
The recursive source snapshot and the actual three-lane proof remain later work.

## Lanes and ownership

| Lane | Location | Input and gate |
| --- | --- | --- |
| 1: projection | Private source repository, fixed self-hosted runner label | Candidate production sources projected into a clean pinned checkout at the old import path; unchanged package tests, fixtures and goldens; inventory and named-test mutation kills. |
| 2: consumer | Separate private workflow, GitHub-hosted Ubuntu | Candidate fetched through a file GOPROXY with a require and no candidate replace; complete consumer CI, converters, composition root and CLI behavior. Z and C1 use an immutable module tag. |
| 3: module | Public repository, GitHub-hosted macOS and Ubuntu | Module vet, tests with race detection, build, and declared platform conformance. This initial workflow tests generic tooling and synthetic module/consumer behavior only. |

The private repository never becomes public through this workflow. No public
job fetches private sources, uses secrets, or selects a self-hosted runner.
`sh1-tooling.yml` grants only `contents: read`, pins Actions, and disables persisted
checkout credentials. GitHub owns both public runners. All validation runs there;
no tests, builds or vet run on the development host.

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
absolute paths and timestamps are excluded. Later private evidence must bind the
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
The private lane-1 acceptance run must have no skips and no name masks.

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
source guard. Later workflows declare converter and real guard mutations too.
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
is not an attestation of a real release. Private workflows derive those values
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

The private gate's evidence bundle must include:

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

## Later private-workflow contract

Lane 1 is a distinct workflow inside the private source repository:

```yaml
on:
  push:
    branches: ['session-host-f/**']
permissions:
  contents: read
jobs:
  projection:
    runs-on: [self-hosted, sh1-projection]
```

There is no `workflow_dispatch`, pull-request event, runner input, or expression
that can change the fixed `sh1-projection` label. The label is a runner class,
not a machine name. No secrets are passed or read. The private Change Request
must configure that fixed class with the runner owner; this public change does
not create or configure a runner.

Check out exact reviewed source and candidate identities into disposable trees.
Run the full package tests recursively (including the recursive provider
subpackage) unchanged, uncached, without skips, and without `-run` masks. Supply
the frozen opt-in environment and tags declared by the package owner. If an
opt-in cannot be exercised without violating the runner's isolation, report that
constraint and stop the gate; do not patch tests or turn a skip into green.
Preserve fixtures and goldens, verify the projection report, publish inventories,
require complete declared guard coverage, and require every declared mutation
to kill its named test. Capture actual shell exits separately from inventories.

Lane 2 is another workflow file in that private repository, on GitHub-hosted
`ubuntu-latest`, with `contents: read` and no secrets. It consumes the identical
candidate identity via a require and the proxy (later the immutable tag), with
no candidate replace, then runs the entire existing consumer CI: configuration,
board, vendor drift, build/document guards, all CLI packages, server and
regression scenarios. Composition, converters and command behavior are real
entry points. It may not replace the whole consumer suite with session subsets.
Both lanes publish evidence keyed to the same source/candidate/transform tuple;
changing any input invalidates the tuple. The public workflow cannot assert
private lane success.

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
| [38015721398](https://github.com/relux-works/curator-session-host/actions/runs/38015721398), initial snapshot | 40 s | 59 s | Both green |
| [38015843173](https://github.com/relux-works/curator-session-host/actions/runs/38015843173), strengthened refusal tests | 31 s | 43 s | Both green |
| [38016057110](https://github.com/relux-works/curator-session-host/actions/runs/38016057110), implementation and workflow verification | 43 s | 62 s | Both green |
| [38019374197](https://github.com/relux-works/curator-session-host/actions/runs/38019374197), revision 2 regressions and narrowing proof | 42 s | 48 s | Both green |

Durations are measured from the hosted jobs' started/completed timestamps,
including setup and artifact upload. All recorded snapshots passed every
public matrix step. These measurements concern synthetic tooling only; private
source/consumer gate durations remain unknown.

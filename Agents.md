# AGENTS.md

## Purpose

This file defines repository instructions for coding agents.

The repository currently has two distinct work modes:

1. **Go test completion mode**
2. **USB/IP over QUIC proxy project mode**

Pick the mode that matches the user task.
If a task spans both modes, apply the project rules first and the test rules second.

---

## Mode selection

### Use Go test completion mode when

- the user asks to fill empty Go table-driven tests
- the user provides existing test skeletons and expects test bodies only
- the task is limited to assertions, mocks, helper functions, and arrange/act/assert logic

### Use USB/IP over QUIC proxy project mode when

- the task is about architecture, implementation, refactoring, documentation, transport, observability, control plane, or cleanup logic for the USB/IP over QUIC proxy
- the task affects runtime code, system behavior, design notes, APIs, or operational logic

### If both apply

- preserve project architecture and compatibility constraints first
- then follow the test-writing constraints for the test code itself

---

# Go test completion mode

## Role

You fill in Go test bodies inside already existing empty table-driven tests.

Tests are in the same package as the production code.

## Goal

Write tests for the expected contract and intended behavior of the SUT, not for its current buggy behavior.

It is acceptable if some tests fail after being filled in. A failing test may indicate a defect in the implementation.

## Hard rules

- Edit only:
  - test bodies
  - helper functions used by tests

- Do not change:
  - `package`
  - imports
  - test names
  - function signatures
  - mock types
  - mock constructors

- Do not add:
  - new types
  - new files
  - new dependencies
  - new interfaces

- Use only existing `gomock` mocks.
- Configure mocks only with:
  - `EXPECT`
  - `Return`
  - `DoAndReturn`

- `gomock.InOrder` is allowed.
- Keep style simple and minimal.
- Do not ask the user clarifying questions. Proceed independently.

## Parallelism

- If the test uses a database (`sqlx`, GORM, Firebird, Postgres testcontainers, and similar), do **not** use `t.Parallel()`.
- If there is no database, `t.Parallel()` is allowed in subtests.

## Behavior policy

- Test cases must describe expected behavior.
- Do not adapt `want` to match a broken implementation.
- Do not weaken assertions just to make tests pass.

## Data style

- Always write struct literals in full form.
- Explicitly include all fields, even when empty:
  - `nil`
  - `""`
  - `0`
  - `false`
  - `uuid.Nil`
  - `sql.NullString{}`
  - empty slices

- Prefer repeating literals across test cases over shared mutable state.
- Do not introduce unnecessary intermediate variables.
- Values such as:
  - `uuid.MustParse("...")`
  - `domain.Resource{...}`
  - `[]string{...}`

  should usually be written inline where used.

## Coverage policy

Cover only meaningful expected scenarios:

- basic success case
- edge cases
- empty inputs
- dependency errors
- database errors
- context cancellation
- normalization like `empty -> nil` if it is part of the contract
- call order if it matters
- panic only if it is realistically implied by the code

Do not add extra cases just for quantity.

## Assertions

Use this style:

```go
if (err != nil) != (tt.wantErr != nil) {
 t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
 return
}
if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
 t.Errorf("errors.Is mismatch: got=%v, want=%v", err, tt.wantErr)
 return
}
if diff := deep.Equal(got, tt.want); len(diff) > 0 {
 t.Error(diff)
}
```

Rules:

- Use variable names:
  - `got`
  - `want`
  - `wantErr`

- Use `t.Fatal` only for critical preconditions.
- Every helper must call `t.Helper()`.

## Test case naming

Use these prefixes:

- `ok/<description>`
- `error/<description>`

Examples:

- `ok/basic`
- `ok/empty_slice`
- `error/invalid_input`
- `error/panic_on_nil`

## Internal decision rule

Before writing code, internally determine:

- what the SUT contract is
- which dependencies are actually needed
- which inputs and expectations are minimally sufficient

Do not overbuild the test.

## Recommended shape

```go
func Test<SUT>_<Method>(t *testing.T) {
 type args struct {
  ctx context.Context
  // other args
 }

 tests := []struct {
  name    string
  args    args
  want    ResultType
  wantErr error
  arrange func(t *testing.T, ctx context.Context, ctrl *gomock.Controller) (deps, error)
 }{
  {
   name: "ok/basic",
   args: args{
    ctx: context.Background(),
    // all fields explicitly filled
   },
   want: ResultType{
    // all fields explicitly filled
   },
   wantErr: nil,
   arrange: func(t *testing.T, ctx context.Context, ctrl *gomock.Controller) (deps, error) {
    t.Helper()

    d := newDeps(ctrl)
    // expectations

    return d, nil
   },
  },
  {
   name: "error/dependency",
   args: args{
    ctx: context.Background(),
    // all fields explicitly filled
   },
   want: ResultType{
    // zero/full fields explicitly filled
   },
   wantErr: ErrExpected,
   arrange: func(t *testing.T, ctx context.Context, ctrl *gomock.Controller) (deps, error) {
    t.Helper()

    d := newDeps(ctrl)
    d.repo.EXPECT().Method(...).Return(..., ErrExpected)

    return d, nil
   },
  },
 }

 for _, tt := range tests {
  tt := tt

  t.Run(tt.name, func(t *testing.T) {
   isDBTest := strings.Contains(tt.name, "db/")
   if !isDBTest {
    t.Parallel()
   }

   ctx := tt.args.ctx
   ctrl := gomock.NewController(t)
   defer ctrl.Finish()

   if strings.HasPrefix(tt.name, "error/panic/") {
    defer func() {
     if r := recover(); r == nil {
      t.Errorf("want panic, got none")
     }
    }()
   }

   d, err := tt.arrange(t, ctx, ctrl)
   if err != nil {
    t.Fatalf("arrange: %v", err)
   }

   got, err := d.sut.Method(ctx /* other args */)
   want := tt.want

   if (err != nil) != (tt.wantErr != nil) {
    t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
    return
   }
   if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
    t.Errorf("errors.Is mismatch: got=%v, want=%v", err, tt.wantErr)
    return
   }
   if diff := deep.Equal(got, want); len(diff) > 0 {
    t.Error(diff)
   }
  })
 }
}
```

## Output rule

Return only one of:

- completed test bodies
- helper function bodies
- this prompt converted into repository instructions

Do not add extra explanation.

Use English comments only where necessary to explain non-obvious logic.

---

# USB/IP over QUIC proxy project mode

## Role

You are a senior systems engineer and implementation agent working on a userspace USB/IP over QUIC proxy.

Your task is to design and implement the project incrementally, preserving compatibility with stock Linux USB/IP components and avoiding unnecessary scope expansion.

## Project

Build a USB/IP over QUIC proxy architecture with two userspace proxies:

- **client-side proxy**
  - accepts local TCP connections from `usbip` / `vhci-hcd`
  - forwards them over QUIC

- **server-side proxy**
  - accepts QUIC streams from client-side proxy
  - opens local TCP connections to `usbipd`
  - manages session lifecycle and zombie cleanup

Target outcome:

- no deep Linux kernel modifications
- compatibility with stock `usbip`, `usbipd`, `usbip-host`, `vhci-hcd`
- the application must be usable as a practical replacement for the legacy `usbip` user workflow via shell aliasing
- user-facing flows should require minimal or no retraining for common operations such as list and attach
- secure transport between proxies
- better behavior on unstable networks
- automatic zombie-session cleanup when the client dies
- observability and operational control

## Non-goals

Do **not** implement or promise:

- kernel modifications
- changes to stock USB/IP wire format on the edges
- shared safe concurrent access to one physical USB device by multiple clients
- transparent session resume after full QUIC connection death
- preservation of attach state across proxy restart
- transparent failover across proxy instances
- changes to ownership semantics of stock USB/IP

## Drop-in replacement requirement

This requirement is critical.

The intended product is not just an internal proxy pair. It should be usable as a practical replacement for the old `usbip` workflow by using a shell alias or equivalent wrapper.

Target operator experience:

- a user should be able to replace the old client command with an alias, wrapper, or drop-in binary
- common commands should remain operationally familiar
- the system should preserve the legacy mental model for `list`, `attach`, and related flows as much as possible
- any unavoidable behavioral differences must be explicit and minimal

Design implications:

- prioritize CLI and workflow compatibility, not only protocol compatibility
- prefer local listening behavior and command surfaces that let existing scripts be adapted with minimal changes
- do not require users to learn a fundamentally new attachment workflow unless explicitly requested
- if a compatibility gap exists, treat it as a product defect or an explicit limitation, not as an acceptable default

## Core architecture

Linux USB/IP remains unchanged at the edges.

Topology:

```text
[ Linux client ]
  usbip / vhci-hcd
        |
        | TCP 127.0.0.1:3240
        v
  client-side proxy
        |
        | QUIC
        v
  server-side proxy
        |
        | TCP 127.0.0.1:3240
        v
  usbipd -> usbip-host -> physical USB device
```

Key rule:

- **1 TCP USB/IP session = 1 QUIC stream**

Rationale:

- `usbip list` usually uses short-lived independent sessions
- `usbip attach` creates a long-lived session carrying URB-related traffic
- byte ordering must be preserved within each session

Therefore:

- do not multiplex bytes from multiple USB/IP TCP sessions into a single QUIC stream
- do not redesign the USB/IP edge protocol
- do not re-encode stock USB/IP PDU formats unless it stays fully internal and invisible at the edges

## Invariants

The implementation must preserve these invariants:

- stock Linux tools should see plain local TCP on both ends
- QUIC is used only between proxies
- ordering of bytes inside one USB/IP session must be preserved
- each accepted TCP session is mapped to a dedicated QUIC stream
- control plane and data plane are separate logical layers
- compatibility is more important than elegance
- explicit cleanup is preferred over optimistic recovery
- when uncertain, choose the simplest implementation that preserves compatibility

## Transport requirements

Implement transport with these properties:

- QUIC between proxies
- TLS 1.3 security
- mutual TLS authentication between proxies
- one long-lived QUIC connection between proxies when possible
- multiple independent streams inside that connection
- keepalive and liveness handling
- tolerance to NAT rebinding and path migration when the same QUIC connection survives

Important:

- path migration is **not** session resume
- if the QUIC connection dies completely, old streams are lost
- after full connection loss, cleanup and fresh attach are required

## Data plane

Responsibilities:

- accept TCP USB/IP session
- map it to one QUIC stream
- proxy bytes in both directions
- preserve order
- handle backpressure
- close stream when TCP session ends
- close TCP session when remote stream ends or fatal transport error occurs

Do **not**:

- parse more USB/IP protocol than needed for routing or session metadata
- introduce protocol behavior not present in stock USB/IP unless it is purely internal to the proxy

## Control plane

Responsibilities:

- session registration
- ownership tracking
- heartbeat and liveness
- state machine for zombie cleanup
- policy enforcement
- observability
- administrative control API

Even if the first prototype only tunnels bytes, keep control plane as a distinct logical subsystem.

## Session model

Server-side proxy maintains a session registry.

Each session should track at minimum:

- `session_id`
- `busid`
- `owner_id`
- `quic_connection_id`
- `quic_stream_id`
- `upstream_tcp_conn`
- `state`
- `attached_at`
- `last_heartbeat_at`
- `grace_deadline`

Optional fields may include:

- `bytes_up`
- `bytes_down`
- `last_transport_error`
- `cleanup_attempts`
- `release_reason`

## Zombie problem

Primary failure scenario:

- client imported a device
- client died or disappeared
- device is gone on the client side
- server-side USB/IP still considers the device busy

Goal:

- server-side proxy must autonomously release the device after confirmed client death

## Liveness model

Use two layers of liveness:

### 1. Transport-level

- QUIC keepalive
- idle timeout
- connection termination detection

### 2. Ownership-level

- explicit heartbeat or control stream from client-side proxy
- used to distinguish temporary transport degradation from owner death

Do not rely on only one signal if a second signal can materially improve cleanup correctness.

## State machine

Implement the session lifecycle conceptually as:

```text
Detached
  -> ImportRequested
  -> Attached
  -> Suspect
  -> GracePeriod
  -> ForceRelease
  -> RebindReady
  -> Detached
```

State intent:

- **Detached**
  - device is free

- **ImportRequested**
  - attach started, ownership not yet stable

- **Attached**
  - active session, heartbeat present, upstream TCP alive

- **Suspect**
  - heartbeat missing or transport failure detected

- **GracePeriod**
  - short wait for transient network degradation

- **ForceRelease**
  - actively clean up stale ownership

- **RebindReady**
  - device cleaned and ready for new attach

## Release policy

Implement cleanup in escalating levels.

### Level 1: soft release

- close upstream TCP session to `usbipd`
- wait for normal USB/IP cleanup

### Level 2: forced release

If soft cleanup did not release the device within timeout:

- terminate server-side export session as needed
- restore device to attachable state

### Level 3: hard cleanup

- `usbip unbind --busid ...`
- `usbip bind --busid ...`
- return device to the available pool

Design cleanup logic so that escalation thresholds are configurable.

## Observability

Provide observability from the beginning.

Minimum required telemetry:

- `session_id`
- `stream_id`
- `busid`
- attach and detach events
- active session count
- transport errors
- cleanup attempts
- cleanup latency
- bytes transferred
- disconnect count
- migration events if detectable

Preferred outputs:

- structured logs
- Prometheus metrics
- tracing hooks
- qlog or equivalent QUIC diagnostics where feasible

## Admin API

Expose a minimal administrative interface.

Recommended endpoints:

- `GET /sessions`
- `GET /sessions/{id}`
- `POST /sessions/{id}/release`
- `POST /devices/{busid}/rebind`
- `GET /metrics`
- `GET /healthz`

If repository conventions differ, adapt the transport but preserve this functional surface.

## Implementation strategy

Work in phases.

### Phase 1: transport-compatible prototype

- implement client-side proxy
- implement server-side proxy
- configure mTLS
- implement stream-per-session mapping
- verify `usbip list` through the proxy
- verify `usbip attach` through the proxy

### Phase 2: baseline observability

- structured logs with `session_id`, `stream_id`, `busid`
- active session counters
- attach and disconnect duration metrics
- error metrics

### Phase 3: zombie cleanup

- session registry
- heartbeat channel
- suspect / grace / force-release flow
- cleanup policy
- rebind workflow

### Phase 4: operational hardening

- admin API
- policy engine
- race-condition protection
- timeouts and limits
- certificate lifecycle handling

## Compatibility rules

Always optimize for compatibility with stock Linux USB/IP behavior and with the existing user workflow around `usbip`.

Preserve unchanged where feasible:

- `vhci-hcd` on client
- `usbipd` on server
- `usbip-host` on server
- wire format at the edges
- `DEVLIST`, `IMPORT`, `SUBMIT`, `UNLINK` semantics at the edges
- expected behavior of the USB stack relative to imported devices
- operator-facing workflow compatibility for common `usbip` commands
- the ability to use the new application through aliasing, wrapping, or a drop-in replacement strategy

If a proposed change risks breaking stock compatibility or makes alias-based replacement materially worse, reject it unless the user explicitly requests a compatibility break.

## Engineering constraints

- do not invent undocumented kernel behavior
- do not assume transparent recovery where transport semantics do not allow it
- do not over-engineer the first version
- prefer a minimal vertical slice that can be tested end to end
- explicitly document assumptions
- if some behavior cannot be guaranteed, say so clearly in code comments and design notes

## Ambiguity policy

If requirements are ambiguous:

- choose the simplest valid interpretation
- do not expand scope with speculative features
- document the chosen assumption in the result

If a detail is missing but a reasonable default exists:

- proceed with that default
- record it under **Assumptions**

Never fabricate exact protocol behavior, kernel semantics, or implementation facts not grounded in the project context or inspected source.

## Research and code reading policy

Before changing code:

- inspect relevant files and trace the current architecture
- identify the minimal set of modules to touch
- avoid broad refactors unless necessary
- prefer incremental edits over rewrites

When interacting with tools:

- be efficient
- avoid noisy exploration
- do not scan unrelated files
- after each write, state exactly what changed and where

## Output contract

For substantial tasks, respond in this format:

1. **Overview**
   - 3-6 sentences max

2. **Assumptions**
   - short bullet list

3. **Plan**
   - short bullet list, ordered

4. **Changes**
   - what changed
   - where
   - why

5. **Risks**
   - only concrete risks

6. **Validation**
   - what was verified
   - what was not verified

7. **Next minimal step**
   - one recommended next step only

For simple tasks:

- answer directly in at most 5 bullets

Do not produce long narrative text.
Do not restate the full prompt back to the user.

## Scope control

Implement exactly and only what is requested in the current task.

- no extra features
- no speculative abstractions
- no hidden UX or API embellishments
- if adjacent improvements are discovered, mark them as optional rather than silently adding them

## Self-check

Before finalizing:

- verify that the design still uses userspace proxies rather than kernel modifications
- verify that `1 TCP session = 1 QUIC stream` remains true
- verify that edge compatibility with stock USB/IP is preserved
- verify that the resulting application can still serve as a practical alias-based replacement for the old `usbip` workflow
- verify that zombie cleanup does not promise impossible transparent session resume
- verify that any unverified claims are labeled as assumptions or limitations

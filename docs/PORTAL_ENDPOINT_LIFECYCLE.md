# Portal selection and endpoint lifecycle

This is the binding-local design contract for issue #6, a partial slice of
parent #4 / Project #2 INT-13. It specifies future Portal selection, persistent
opaque endpoint identity, and fabric lifecycle responsibilities. It does not
implement a Portal, persistence store, launcher, or production endpoint, and
does not complete the parent acceptance criteria.

## Current boundary and ownership

The binding owns SDK integration, node and endpoint allocation, commissioning,
fabric-storage integration, and reporting. Protocol-neutral semantics and Matter
target mappings remain with their public owners identified by the
[reviewed lock](../sdk/matter-sdk.lock.json) and [repository overview](../README.md).
Native protocol facts must first be documented in the public native contract
and accepted through the semantic evidence path; Portal selection cannot invent
or upgrade them. This document introduces no protocol facts or target mappings.

The existing [readiness evaluator](../runtime/production_readiness.py) is an
evidence decision only. It creates no node, endpoint, or fabric. Today every
invocation denies allocation: the reviewed lock has no accepted runtime
composition-profile digest. Selected capabilities must therefore be shown as
**withheld**, with the evaluator's denial reason. There is no commissioning or
endpoint allocation path in this revision.

“Stable endpoint identity” here means persistent identity across restarts within
the same binding node lifecycle. It is not an API stability promise. The design
and interfaces remain pre-stable before 1.0; no MCP namespace or GraphQL contract
changes are introduced here.

## Selection and visible state

Portal selection records publication intent per semantic capability, not proof
of target eligibility or permission to commission. Selection must be explicit;
capability discovery alone must not publish anything. The following states are
design terms, not existing evaluator return fields or an API schema:

| State | Meaning | Endpoint and fabric effect |
| --- | --- | --- |
| Unselected | Publication is not requested. | No active endpoint allocation, commissioning, or fabricated fabric material. |
| Withheld | Publication is requested but evidence or safe restoration is insufficient. | No active endpoint allocation or commissioning; expose the denial reason. |
| Eligible | A future reviewed target and matching evidence pass admission. | Intent remains pending until future runtime lifecycle requirements are met. |
| Published | Future runtime has admitted and safely restored or created the endpoint. | Reporting is limited to the selected, evidenced target contract. |

Persist selection intent separately from endpoint and fabric state. A restart,
retry, UI refresh, or incoming value must not turn withheld into published.
Unselection stops publication and reporting and removes any active endpoint in a
future runtime. It must not erase native evidence or partial last-known-good
semantic state, nor present retained values as fresh measurements. Deselection
and evidence failure must not silently select a substitute target.

## Readiness trace and prerequisites

A future launcher must call `evaluate(lock, composition, build_evidence)` before
any allocation. Portal controls cannot bypass this decision. Trace the contract
to the existing functions as follows:

| Evaluator condition | Consequence for selection |
| --- | --- |
| `validate_lock` requires equality with the reviewed baseline. | Drift is withheld, including a caller-edited profile pin. |
| `evaluate` rejects the absent accepted profile pin. | All currently selected capabilities are withheld, regardless of other inputs. |
| A future reviewed pin must match `_composition_digest`. | A self-asserted accepted profile cannot substitute for the reviewed profile. |
| `validate_build_evidence` requires the matching SDK identity and all required results. | Missing, mismatched, malformed, or failed build evidence cannot permit allocation. |
| `validate_composition` requires the accepted profile shape and matching SDK, mapping, and device type. | A proposed template or different target remains withheld. |
| Required composition facts need evidence. | Missing Descriptor, Power Topology, or Electrical Power Measurement facts cannot be supplied by defaults or SDK sample metadata. |

The required build results are `all_devices_app`, `chip_tool`,
`electrical_power_measurement_tests`, `power_topology_tests`, and
`upstream_electrical_sensor_fixture`. Mandatory Electrical Power Measurement
facts include non-unknown `PowerMode`, positive `NumberOfMeasurementTypes`, a
non-empty `Accuracy` list matching that count with evidenced entries, and an
evidenced nullable `ActivePower` fact. Optional `ActiveCurrent` alone is
insufficient. The [profile template](../examples/runtime-composition.profile.json)
is proposed and does not admit a target.

The target-profile decision must precede runtime implementation: obtain accepted
native/semantic evidence for the missing composition, choose another
standards-valid evidenced target through review, or continue withholding. A
future reviewed lock and admissible matching build evidence are prerequisites;
this document changes neither. The validator checks evidence structure and
references; it does not independently prove the truth of supplied native facts.
Evidence labels Proven / Hypothesis / Unknown remain separate from native
qualification and target admission. Selection does not upgrade either axis.

## Persistent opaque identity design

A future admitted runtime must persist a locally generated opaque binding
identity associated with the semantic capability, target composition, binding
node identity, and assigned Matter endpoint number. Portal labels, discovery
order, transient values, and device-family display names are not identity keys.
Public examples use synthetic families and must contain no real identifiers.

Within the same node lifecycle, restart and deselect/reselect must recover the
same identity and endpoint number for the same compatible composition. Retained
inactive identity records are bookkeeping, not allocated endpoints. Unselected
or withheld capabilities without a prior admitted endpoint receive no endpoint
number or placeholder allocation. Retired numbers must not be assigned to a
different capability within that node lifecycle.

Persist the identity-to-endpoint association durably before exposing an endpoint;
restore and validate it before reporting or commissioning. Missing, corrupt,
ambiguous, or conflicting persisted state must withhold affected publication
without guessing a new identity. Preserve valid unaffected records and native
last-known-good state. A composition or node-identity change requires an explicit
reviewed migration or retirement decision; it cannot silently reuse an old
identity for a different target. Atomic storage, recovery, and migration tests
remain future runtime work.

## Fabric lifecycle design

Endpoint identity is scoped to a binding node and does not encode credentials,
fabric identifiers, network coordinates, or controller identity. Fabric storage
belongs to the node's SDK lifecycle and is separate from Portal selection and
semantic caches. No selection, withholding, synthetic fixture, or accepted build
may fabricate commissioned state, credentials, or fabric material.

For a future admitted runtime, commissioning also requires explicit operator
confirmation at action time under [the repository safety rules](../AGENTS.md).
This is a real-fabric safety requirement, not a new Portal authorization gate.
No real commissioning is authorized by this design. Authentic existing fabric
material must use protected runtime storage and must never enter this repository,
reports, or public fixtures.

Deselecting or withholding one endpoint stops its publication but must not erase
legitimate node-level fabric state shared with other endpoints. It creates no
new fabric material. A restart may restore genuine existing fabric state only
through future validated SDK integration; it must not infer commissioning from a
selection flag or cached value. Removing a fabric follows the SDK node lifecycle
and must not manufacture a replacement or automatically rejoin. Removing the last
fabric need not erase the local endpoint identity ledger; any later commissioning
still requires admission and action-time confirmation. An explicitly confirmed
node reset retires the old node lifecycle and clears its owned fabric state and
identity ledger; continuity after reset must not be claimed. These are design
requirements, not implemented or exercised reset/commissioning operations.

## Offline lifecycle scenarios

| Scenario | Required outcome |
| --- | --- |
| Discover an unselected capability. | No endpoint or fabric material is created. |
| Select a capability with today's reviewed lock. | Withheld with the absent-profile reason; no allocation or commissioning. |
| Supply accepted-looking metadata and passing build JSON today. | Still withheld; immutable reviewed baseline controls admission. |
| Restart with selection intent but no previously admitted endpoint. | Restore intent and recompute withholding; no placeholder endpoint. |
| Future admitted endpoint restarts with valid persistence. | Re-evaluate evidence and restore the same identity and endpoint number before exposure. |
| Future admitted endpoint is deselected, then selected again. | Remove active publication; retain inactive identity; reuse only after renewed admission with compatible composition. |
| Evidence becomes invalid while other endpoints remain valid. | Withhold the affected publication; preserve valid records, native evidence, and partial last-known-good state. |
| Persistence is corrupt or target composition changes. | Withhold affected publication pending recovery or reviewed migration; no guessed identity. |
| One endpoint is withheld on a node with legitimate fabrics. | No new fabric state and no deletion of shared legitimate material. |
| A fabric is removed or a node reset is explicitly requested. | Follow the distinct lifecycle rules above; never automatically commission or rejoin. |

These scenarios are an offline design review, not runtime lifecycle tests.
The [SDK fixture procedure](SDK_BASELINE.md) demonstrates only a bounded upstream
sample; it cannot establish production admission, interoperability, certification,
physical qualification, or parent #4 completion. Future runtime work must provide
persistence/recovery and SDK integration tests alongside full admissible target
evidence. Physical and live-fabric smoke remain outside this design deliverable.

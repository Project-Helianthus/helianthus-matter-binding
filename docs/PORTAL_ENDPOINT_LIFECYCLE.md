# Publication rule and endpoint lifecycle

This is the binding-local design contract for issue #6, a partial slice of
parent #4 / Project #2 INT-13, as changed by issue #121. It specifies which
live resources the bridge publishes, endpoint numbers keyed to canonical
resources, and fabric lifecycle responsibilities. It does not implement a
Portal, launcher, or production endpoint, and does not complete the parent
acceptance criteria.

## Current boundary and ownership

The binding owns SDK integration, node and endpoint allocation, commissioning,
fabric-storage integration, and reporting. Protocol-neutral semantics and Matter
target mappings remain with their public owners identified by the
[reviewed lock](../sdk/matter-sdk.lock.json) and [repository overview](../README.md).
Native protocol facts must first be documented in the public native contract
and accepted through the semantic evidence path; the publication rule cannot invent
or upgrade them. This document introduces no protocol facts or target mappings.

The existing [readiness evaluator](../runtime/production_readiness.py) is an
evidence decision only. It creates no node, endpoint, or fabric. Today every
invocation denies allocation: the reviewed lock has no accepted runtime
composition-profile digest. Every resource the publication rule would publish
must therefore be shown as **withheld**, with the evaluator's denial reason.
There is no commissioning or endpoint allocation path in this revision.

“Stable endpoint number” here means a number that follows its resource across
restarts within the same binding node lifecycle. It is not an API stability promise. The design
and interfaces remain pre-stable before 1.0; no MCP namespace or GraphQL contract
changes are introduced here.

## Publication rule and visible state

There is no per-capability selection and no per-resource switch. One rule
decides, implemented by the [Go publication package](../publication/rule.go)
and the [Python publication rule](../runtime/publication_rule.py):

- A live resource is published only when its catalog element has the
  disposition `transformed` on the derived Matter roster and that element is
  not named in the exclusion list.
- `unknown`, `unsupported` and `withheld` rows, a row without a disposition, an
  element without a roster row and an empty or malformed resource id are not
  published. Each is withheld with a named reason and gets no number.
- Several transformed elements of one resource are one endpoint, not one
  endpoint per element. One element on two resources is two endpoints.
- A relation allocates no endpoint.

The exclusion list is [`publication/exclusions.json`](../publication/exclusions.json),
one JSON object with one array `elements`. The committed array is empty. The
Go package embeds that file and the Python rule reads the same file; there is
no second list. A malformed list is refused. The rule adds no mapped row and
never upgrades a roster disposition.

The following states are design terms, not existing evaluator return fields or
an API schema:

| State | Meaning | Endpoint and fabric effect |
| --- | --- | --- |
| Withheld | The rule does not publish the element, or evidence or safe restoration is insufficient. | No active endpoint allocation or commissioning; expose the named reason. |
| Eligible | The rule publishes the resource and a future reviewed target and matching evidence pass admission. | The number is stored; exposure waits for future runtime lifecycle requirements. |
| Published | Future runtime has admitted and safely restored or created the endpoint. | Reporting is limited to the evidenced target contract. |

A restart, retry, UI refresh, or incoming value must not turn withheld into
published. A resource that leaves the target report stops publication and
reporting in a future runtime. It must not erase native evidence or partial
last-known-good semantic state, nor present retained values as fresh
measurements. Evidence failure must not silently publish a substitute target.

## Readiness trace and prerequisites

A future launcher must call `evaluate(lock, composition, build_evidence)` before
any allocation. Portal controls cannot bypass this decision. Trace the contract
to the existing functions as follows:

| Evaluator condition | Consequence for publication |
| --- | --- |
| `validate_lock` requires equality with the reviewed baseline. | Drift is withheld, including a caller-edited profile pin. |
| `evaluate` rejects the absent accepted profile pin. | Every resource the rule would publish is withheld, regardless of other inputs. |
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
qualification and target admission. The publication rule does not upgrade
either axis.

## Resource-keyed endpoint ledger

The [endpoint selector](../runtime/endpoint_selection.py) keys every Matter
endpoint number to a canonical resource. The identity and relation rules are
those of the graph contract in helianthus-docs-semantic (`api/v1/graph.md`,
sections 6, 7, 32 and 33); this repository does not decide them.

Offers come from the Matter target report over a named lens. Each offer is one
disposition that offers a catalog element on exactly one resource, cites that
resource's `ResourceID` and anchor key, and cites the relation keys it was made
from. The selector asks the publication rule about every offer before its
resource can become eligible. A second offer of the same element on the same
resource, or a malformed offer, is withheld. A report that is absent or
malformed publishes nothing and is reported as a fault.

One endpoint maps to one resource (CID-1): elements offered on the same
resource share its endpoint. A ledger record is exactly
`{resource_id, composition_sha256, endpoint}` for the binding node lifecycle
named by the ledger. The anchor, the element, Portal labels, discovery order,
transient values, and device-family display names are not ledger keys. Public
examples use synthetic families and must contain no real identifiers.

The endpoint number follows the `ResourceID` (CID-3):

| Event | `ResourceID` | Endpoint number |
| --- | --- | --- |
| Restart, or the resource leaves the report and returns | same | same record, same number |
| Anchor supersession | same | same record, same number |
| Equipment replaced by another unit | new | new record and next number; the old record stays |
| Function kept across an equipment replacement | same | same record, same number |
| Composition profile changed | same | withheld; the record stays pending a reviewed migration |

Endpoint numbers survive restart and are never reused. Within the same node
lifecycle a number is stored before the endpoint may be exposed, the same
`ResourceID` always returns the same number, and a number is never given to
another `ResourceID`. Absence frees nothing, so the number of a replaced unit is
never reused. The Go store assigns the lowest number no stored id holds, from 1
upward; the Python ledger assigns the next number above the highest recorded
one. Neither frees a number, so a store either of them wrote has no gap and
both give the same sequence. The store holds 64 resource ids for the life of
the node; a further new id is withheld with a named reason and the stored ids
are unchanged. Save writes one complete document; a malformed document is
refused and allocates nothing. Retained inactive records are bookkeeping, not
allocated endpoints. A withheld element without a prior admitted endpoint
receives no number or placeholder allocation.

Parent and child endpoints follow the cited relations (TGT-2). On every
reconciliation an eligible resource is placed under the endpoint of the `from`
resource of a relation instance that an offer cites with this resource as `to`,
when that instance is published in the report exactly once under the same
`(pack, pack version, definition, version, from, to)` key, is `coherent`, and
has the forest core type `contains` or `hosts`. The endpoint is presented under
no parent, with a stated reason, when a cited relation to it is not such an
instance, when cited relations name more than one parent, when the parent has
no eligible endpoint, or when the parents form a cycle. `groups`, `measures`,
`supplies`, `controls` and `relays` instances make no parent. Parentage is
never persisted (CID-5): a changed relation moves the endpoint without
rewriting the ledger.

The selector never reads the Portal catalog (PTL-5). Withdrawing a Portal
contribution leaves the ledger and every reconciliation result unchanged. A
resource that leaves the target report keeps its record, so the same number
returns with the resource. Taking a resource off the bridge does not delete any
other file: the ledger, other resources' records and fabric storage stay as
they are.

Persist a new record durably before exposing its endpoint; restore and
validate it before reporting or commissioning. A failed write exposes neither
the endpoint nor a parent placement under it. Missing, corrupt, ambiguous, or
conflicting persisted state withholds the affected resources without guessing
a number: a resource with two records, two resources with one number, or a
malformed record, which also withholds every new allocation. Preserve valid
unaffected records and native last-known-good state. A composition or
node-identity change requires an explicit reviewed migration or retirement
decision outside this module; the ledger accepts only the record shape above.

## Fabric lifecycle design

The endpoint ledger is scoped to a binding node and does not encode credentials,
fabric identifiers, network coordinates, or controller identity. Fabric storage
belongs to the node's SDK lifecycle and is separate from the publication rule and
semantic caches. No publication decision, withholding, synthetic fixture, or
accepted build may fabricate commissioned state, credentials, or fabric material.

For a future admitted runtime, commissioning also requires explicit operator
confirmation at action time under [the repository safety rules](../AGENTS.md).
This is a real-fabric safety requirement, not a new publication gate.
No real commissioning is authorized by this design. Authentic existing fabric
material must use protected runtime storage and must never enter this repository,
reports, or public fixtures.

Taking a resource off the bridge or withholding one endpoint stops its
publication but must not erase legitimate node-level fabric state shared with
other endpoints. It creates no new fabric material. A restart may restore
genuine existing fabric state only through future validated SDK integration; it
must not infer commissioning from a publication decision or cached value. Removing a fabric follows the SDK node lifecycle
and must not manufacture a replacement or automatically rejoin. Removing the last
fabric need not erase the local endpoint ledger; any later commissioning
still requires admission and action-time confirmation. An explicitly confirmed
node reset retires the old node lifecycle and clears its owned fabric state and
endpoint ledger; continuity after reset must not be claimed. These are design
requirements, not implemented or exercised reset/commissioning operations. A
node reset is not part of the publication rule slice; the store has no reset
or delete operation.

## Offline lifecycle scenarios

| Scenario | Required outcome |
| --- | --- |
| Discover an element whose roster row is not `transformed`, or that has no row. | Withheld with its named reason; no endpoint or fabric material is created. |
| Discover an element named in the exclusion list. | Withheld as excluded; no number. |
| Discover a transformed element with today's reviewed lock. | Withheld with the absent-profile reason; no allocation or commissioning. |
| Supply accepted-looking metadata and passing build JSON today. | Still withheld; immutable reviewed baseline controls admission. |
| Restart with no previously admitted endpoint. | Recompute withholding; no placeholder endpoint. |
| Future admitted endpoint restarts with valid persistence. | Re-evaluate evidence and restore the same resource record and endpoint number before exposure. |
| Future admitted resource leaves the report, then returns. | Remove active publication; retain the inactive record; reuse only after renewed admission with compatible composition; delete no file. |
| A new resource arrives when the store already holds 64 ids. | Withheld; stored ids unchanged. |
| The resource's anchor is superseded. | Same `ResourceID`, same record and endpoint number. |
| The equipment is replaced by another unit. | New `ResourceID`, new record and next endpoint number; the old number is never reused. |
| The cited relations change. | The endpoint moves to its new parent or to no parent; the ledger is not rewritten. |
| A Portal contribution is withdrawn. | Ledger and reconciliation results unchanged. |
| Evidence becomes invalid while other endpoints remain valid. | Withhold the affected publication; preserve valid records, native evidence, and partial last-known-good state. |
| Persistence is corrupt or target composition changes. | Withhold affected publication pending recovery or reviewed migration; no guessed number. |
| One endpoint is withheld on a node with legitimate fabrics. | No new fabric state and no deletion of shared legitimate material. |
| A fabric is removed or a node reset is explicitly requested. | Follow the distinct lifecycle rules above; never automatically commission or rejoin. |

These scenarios are an offline design review, not runtime lifecycle tests. The
publication package and the endpoint selector's offline tests exercise the rule,
the store, the ledger and the composition rows with a fake store, a synthetic
roster and a synthetic target report.
The [SDK fixture procedure](SDK_BASELINE.md) demonstrates only a bounded upstream
sample; it cannot establish production admission, interoperability, certification,
physical qualification, or parent #4 completion. Future runtime work must provide
persistence/recovery and SDK integration tests alongside full admissible target
evidence. Physical and live-fabric smoke remain outside this design deliverable.

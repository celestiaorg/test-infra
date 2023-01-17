# Test Plan #001: Big Blocks Creation/Sync

- [Test Plan #001: Big Blocks Creation/Sync](#test-plan-001-big-blocks-creationsync)
  - [Introduction](#introduction)
  - [In-Scope](#in-scope)
  - [Out-of-Scope](#out-of-scope)
  - [Risks](#risks)
  - [Entry Conditions](#entry-conditions)
  - [Exit Conditions](#exit-conditions)
  - [Test Environment](#test-environment)
  - [Notes](#notes)
  - [Test-Cases](#test-cases)

## Introduction

The motivation behind this plan is to test how our stack(celestia-core/celestia-app/celestia-node) can withstand the max peak usage of the network from submitting data into our DA/Consensus Layer.
In addition, this plan is covering cases where we will have different implementations of the stack(i.e. celestia-core/celestia-app/celestia-node are developed in Rust, etc.) and the plan should 
be applied for them, too as we are having for current implementation(Go)

## In-Scope

- Max 8 MB block size
- Celestia App Instances
  - We are covering here only `validator` mode
  - 40/100 validators’ set
- Celestia Node Instances
  - Bridge / Full / Light
- Network Latencies / Chaos
- Chain up to 500 blocks (\*)
- Implementation of either of the stack in different language

## Out-of-Scope

- Rollmint submitting data using public api or any other way
- Malicious behaviour from the validators’ set
- Withholding the data
- Losing/Restoring connection between peers
- Gas fees for submitting data(\*\*)

## Risks

- This plan is not covering the sync time for a long-live chain, which might uncover further defects before start of incentivised testnet
- From an economic standpoint, we need to be aware of the costs that the users will bear if the block space is too scarce and how much premium should be payed to get the data included in the block

## Entry Conditions

- Sync between DA Network and underlying Consensus network is happening in a baseline test-case(core/app can produce empty blocks and node can sync/propagate them)(\*\*\*)
  - e.g. not a dependency upgrade/downgrade issue in the code-base
- Reporting of any encountered defects during execution of this test-plan
- No blocking issues before execution of this test-plan

## Exit Conditions

- All In-Scope testing has been done
- No unresolved encountered Critical or High level defects
- Medium to Low level defects are documented in respective repos
- Test Report is presented

## Test Environment

- Testground
- K8s cluster
- Metrics' dashboards

## Notes

[E2E: Celestia Network Tests](https://github.com/celestiaorg/celestia-node/issues/7)

[DASing with different max block sizes](https://github.com/celestiaorg/celestia-node/issues/266)

(\*) - Considering that we have 30 seconds block time, and we want to test on the span of 500 blocks, the test run will be around 4 hours

(\*\*) - We make an assumption that all wallets have more than enough money to cover all the costs of submitting txs

## Test-Cases

[Test-Case #001 - Validators submit large txs](test-cases/tc-001-val-large-txs.md)

[Test-Case #002 - DA nodes are in sync with validators’ ](test-cases/tc-002-da-sync.md)

[Test-Case #003 - Full nodes are syncing past headers faster than validators produce new ones](test-cases/tc-003-full-sync-past.md)

[Test-Case #004 - Full and Light nodes are syncing past headers faster than validators produce new ones](test-cases/tc-004-full-light-past.md)

[Test-Case #005 - Light nodes are DASing past headers faster than validators produce new ones](test-cases/tc-005-light-past.md)

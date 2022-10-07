# Test Plan #001: Big Blocks Creation/Sync

- [Test Plan #001: Big Blocks Creation/Sync](#test-plan-001-big-blocks-creationsync)
  - [Introduction](#introduction)
  - [In-Scope](#in-scope)
  - [Out-of-Scope](#out-of-scope)
  - [Entry Conditions](#entry-conditions)
  - [Exit Conditions](#exit-conditions)
  - [Test Environment](#test-environment)
  - [Notes](#notes)
  - [Test-Cases](#test-cases)

## Introduction

The motivation behind this plan is to test how our node implementations (celestia-node) behaves at max peak usage when participating in Data Availability Sampling.


## In-Scope

- Max 4 MB block size
- Celestia Node Instances
  - Bridge / Full / Light
- Network Latencies / Chaos
- DASing Latest Head

## Out-of-Scope

- Eclipse Attacks
- and Network Outages
- Withholding Data Attacks
- Bad Erasure Coding Attacks

## Entry Conditions

- Bridge or Full Node must have enough of blocks up to any height (height 1 is also acceptable) to have a latest head to DAS

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

[DASing with different max block sizes](https://github.com/celestiaorg/test-infra/issues/83)


## Test-Cases

[Test-Case #001 - Light Nodes Must Finish DASing before Block Time](test-cases/tc-001-ln-finish-das-before-block-time.md)

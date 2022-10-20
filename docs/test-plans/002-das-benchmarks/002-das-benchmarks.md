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
The goal of this test plan is to benchmark our Full and Bridge Node implementation against X amount of light nodes to measure performance at peak usage when participating in Data Availability Sampling.

## In-Scope
- Celestia Node Instances
  - Bridge / Full / Light
- Max 4 MB block size
- Max Share Size: 128
- Network Latencies between 60 and 300ms
- DASing will concern both:
  - Latest HEAD
  - a few random sampling ranges

## Out-of-Scope

- Eclipse Attacks
- Network Outages
- Withholding Data Attacks
- Bad Erasure Coding Attacks

## Entry Conditions

- Bridge or Full Node must have enough of blocks up to any height depending on the test case.
  - For latest HEAD test cases, a height of 1 is acceptable
  - For sampling ranges test cases, at least a minimum of height=50

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

[Benchmarking Full And Bridge Nodes against X Light Nodes]](https://github.com/celestiaorg/test-infra/issues/83)


## Test-Cases

[Test-Case #001 - Light Nodes Must Finish DASing before Block Time](test-cases/tc-001-x-light-finish-das-before-block-time.md)

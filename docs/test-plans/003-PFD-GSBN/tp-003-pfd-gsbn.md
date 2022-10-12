# Test Plan #003: DA Nodes can submit PFD and get shares by namespaces

- [Test Plan #003: DA Nodes can submit PFD and get shares by namespaces](#test-plan-003-da-nodes-can-submit-pfd-and-get-shares-by-namespaces)
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

The motivation behind this plan is to test how celestia node types can submit data(also called as PFD - pay for data) to underlying tendermint p2p stack as well as get the data from it(or gsbn - get shares by namespace)
This plan is an extension plan of [TP#001 - Big Blocks](../001-Big-Blocks/tp-001-big-blocks-creation-sync.md). 

## In-Scope

- Max 4 MB block size
- Celestia App Instances
  - We are covering here only `validator` mode
  - 40/100 validators’ set
- Celestia Node Instances
  - Bridge / Full / Light can submit `pfd`
  - - Bridge / Full / Light can `gsbn`
- Network Latencies / Chaos
- Chain up to 500 blocks
- Implementation of either of the node in different language(s)

## Out-of-Scope

- Optimint submitting data using public api or any other way
- Negative cases(i.e. pfd with insufficient balances)
- Losing/Restoring connection between peers
- Gas fees for submitting data(\*)

## Risks

- This plan is not covering the sync time for a long-live chain, which might uncover further defects before start of incentivised testnet
- From an economic stand point, we need to be aware of the costs that the users will bear if the block space is too scarce and how much premium should be payed to get the data included in the block
- At the moment of writing, pfds are synchronise calls only

## Entry Conditions

- TP#001 is not populating any severe defects
  - e.g. we can sync big blocks 
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

[EPIC: DA Nodes submit PFD and GetSharesByNamespace](https://github.com/celestiaorg/test-infra/issues/85)

[EPIC: Test Plan #001 - Big blocks](https://github.com/celestiaorg/test-infra/issues/77)

(\*) - We make an assumption that all wallets have more then enough money to cover all the costs of submitting txs

## Test-Cases

Bridge Node can submit N Amount PFDs -> Full / Light Node can get shares by namespace
Full Node can submit N Amount PFDs -> Bridge / Light Node can get shares by namespace
Light Node can submit N Amount PFDs -> 

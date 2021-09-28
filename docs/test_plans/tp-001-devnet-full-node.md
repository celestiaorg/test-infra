# Test Plan #001: Devnet Celestia Full Node Setup

## Introduction

For Devnet stage, Celestia Full node can have several setup options to communicate with Celestia Core. The latter is a tendermint consensus node. For further text, we will refer to acronyms

- Celestia Full Node = CFN
- Celestia Core Node = CCN

Setup options for CFN can be described with this list:

1. `default` flag, which embeds CCN
2. `--core.disable` flag, which excludes CCN
3. `--core.remote` flag, which connects to an external CCN

## In-Scope

Focus on CFN setup options

- Integration Testing for CFN default/remote CCN setups

## Out-of-Scope

- CFN with disabled CCN (will be covered in #7)
- Light clients
- Performance testing scenario N->1 for CFN->CCN

## Risks

- This is unsufficient to cover the cases where there are more CFNs with different setups interacting with each other
- This plan is not covering the scenarios of constant unstable connection to CCN for remote CFN

## Entry Conditions

- Existing unit tests for CFN and CCN should be green
- No known Critical or High level defects before execution
- Reporting of any encountered defects during execution

## Exit Conditions

- All In-Scope testing has been done
- No unresolved encountered Critical or High level defects
- Actionable items/decisions has been made by the community on Medium to Low level defects
- Test Report is presented

## Timescales

> This section contains estimations for completing the test plan

## People

> This section points who are going to participate in this test plan

## Test Environment

> This section stores information on tools and environment used to execute tests on

## Notes

[ADR #002: Devnet Celestia Core <> Celestia Node Communication](https://github.com/celestiaorg/celestia-node/blob/main/docs/adr/adr-002-predevnet-core-to-full-communication.md)

[Celestia Core Repo](https://github.com/celestiaorg/celestia-core)

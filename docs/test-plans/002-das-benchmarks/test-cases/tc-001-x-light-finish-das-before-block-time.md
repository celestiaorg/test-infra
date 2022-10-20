# Test-Case #001 - Light Node Must Finish DASing in less than block time

## Pre-Requisites:

1. The Full Node has the latest head
2. All light nodes are network-bootstrapped and connected to the full node (no discovery required)
3. Share size is 32

## Steps for each of the light nodes:

1. Sets up network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Performs DAS queries for latest head
   1. `Y` times

## Data Set:

| Number of Light Nodes<br />`I` |            Bandwidth / Latency per Light Node<br />`J`             ` | Sample amount<br />`Y` |
| :---------------------------: | :---------------------------------------------------------------: | :--------------------: |
|              280               | 1. 12MiB / 60ms <br /> 2. 4MiB / 100ms <br /> 3. 50MiB / 30ms <br /> 4. 20MiB / 200ms |             20           |
|              500               | 1. 12MiB / 60ms <br /> 2. 4MiB / 100ms <br /> 3. 50MiB / 30ms  <br /> 4. 20MiB / 200ms  |             20           |
|              800               | 1. 12MiB / 60ms <br /> 2. 4MiB / 100ms <br /> 3. 50MiB / 30ms  <br /> 4. 20MiB / 200ms |             20           |
|              1000               | 1. 12MiB / 60ms <br /> 2. 4MiB / 100ms <br /> 3. 50MiB / 30ms   <br /> 4. 20MiB / 200ms |             20           |


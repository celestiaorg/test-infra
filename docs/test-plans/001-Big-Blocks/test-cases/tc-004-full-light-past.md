# Test-Case #004 - Full and Light nodes are syncing past headers faster then validators produce new ones

## Pre-Requisites:

1. Every validator has enough funds in the account
2. The chain has created the first block
3. Every validator has enough peers
4. Validators’ set is not changing during test execution
   1. All validators are created during genesis
5. DA Nodes amount is not changing during the test execution
   1. We add DA Nodes as the first block is produced

## Steps for each of the validators:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Generates and broadcasts
   1. `X` kb of random data
   2. `Y` times
   3. IP and Genesis Hash for DA Bridge nodes
3. Checks the block size is bigger then 3.5 MiB

## Steps for each of the DA Bridge nodes:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Connects to respective Validator
3. Shares the genesis hash and ip to Full / Light Nodes
4. Check that it is synced
5. Broadcasts new blocks to the DA network

## Steps for each of DA Full / Light nodes:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Receives the trusted genesis hash and ip from Bridge Nodes
3. Waits until `N` amount of block has been produced by the chain
4. Starts syncing the chain afterwards
5. Light checks that it can:
   1. DASes the past headers faster then new blocks are produced (\*)
6. Full checks that it can:
   1. Sync the past headers faster then new blocks are produced

## Data Set:

| Number of Validators / Bridges / Fulls / Lights <br /> `I` |                                Bandwidth / Latency per v/b/f/l <br /> `J`                                | KB of random data <br />`X` | Submit amount <br />`Y` | Amount of Past Blocks <br />`N` |
| :--------------------------------------------------------: | :------------------------------------------------------------------------------------------------------: | :-------------------------: | :---------------------: | :-----------------------------: |
|                     40 / 40 / 20 / 100                     | 1. 256(v/b/f)-100(l)MiB / 0ms <br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms |             100             |           50            |               30                |
|                   100 / 100 / 50 / 1000                    | 1. 320(v/b/f)-100(l)MiB / 0ms<br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms  |             40              |           100           |               50                |

## Notes:

(\*) - We need to measure the DAS of past headers time to have a baseline for further benchmarking of the new p2p stack that is implemented

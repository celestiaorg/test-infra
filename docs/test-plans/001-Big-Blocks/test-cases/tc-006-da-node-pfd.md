# Test-Case #006 - All DA nodes can submit data

## Pre-Requisites:

1. Every validator has enough funds in the account
2. The chain has created the first block
3. Every validator has enough peers
4. Validators’ set is not changing during test execution
   1. All validators are created during genesis
5. DA Nodes amount is not changing during the test execution
   1. We gracefully add DA Nodes as the first block is produced

## Steps for each of the validators:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Generates and broadcasts
   1. IP and Genesis Hash for DA Bridge nodes
3. Checks the block size is bigger then 3.5 MiB

## Steps for each of the DA nodes:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Bridge nodes connect to respective Validators
3. Full / Light nodes connect to Bridge Nodes
4. Checks that the latest received height is the same as for the validators
5. Generates and broadcasts (\*) (\*\*) (\*\*\*)
   1. `X` kb of random data
   2. `Y` times

## Data Set:

| Number of Validators / Bridges / Fulls / Lights <br /> `I` |                                Bandwidth / Latency per v/b/f/l <br />`J`                                | KB of random data <br />`X` | Submit amount <br />`Y` |
| :--------------------------------------------------------: | :-----------------------------------------------------------------------------------------------------: | :-------------------------: | :---------------------: |
|                     40 / 40 / 20 / 100                     | 1. 256(v/b/f)-100(l)MiB / 0ms<br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms |             25              |           10            |
|                   100 / 100 / 50 / 1000                    | 1. 320(v/b/f)-100(l)MiB / 0ms<br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms |             3.5             |           10            |

## Notes:

(\*) - Here All Bridge/Full/Light nodes are submitting data

(\*\*) - In the future cases we explore how will the chain work if Light/Full posts data

(\*\*\*) - It is also interesting to see what will happen to a chain if only 4-5 big TXs (aprox 500kb each) are submitted by light nodes

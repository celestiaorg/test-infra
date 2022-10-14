# Test-Case #003 - Full Nodes post pfd and Light get them by namespace id

## Pre-Requisites:

1. Every validator has enough funds in the account
2. The chain has created the first block
3. Every validator has enough peers
4. Validators’ set is not changing during test execution
   1. All validators are created during genesis
5. DA Nodes' accounts are funded

## Steps for each of the validators:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Generates and broadcasts
   1. IP and Genesis Hash for DA Bridge nodes
3. Checks the tx amount is the same as DA Nodes amount

## Steps for Bridge DA nodes:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Bridge nodes connect to respective Validators

## Steps for Full DA nodes

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Full nodes connect to Bridge Nodes
3. Starts syncing the chain
4. Generates PFD wtih
   1. `X` kb of random data
   2. `Y` amount of namespace id
   3. `Z` times

## Steps for Light DA nodes

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Full nodes connect to Bridge Nodes
3. Starts syncing the chain
4. Gets shares by namespace id
5. Checks that data is the same as full nodes has submitted

## Data Set:

| Number of Validators / Bridges / Fulls / Lights <br /> `I` |                                Bandwidth / Latency per v/b/f/l <br /> `J`                                | KB of random data <br /> `X` | Amount of namespace id <br /> `Y` | Times <br/> `Z` |
| :--------------------------------------------------------: | :------------------------------------------------------------------------------------------------------: | :--------------------------: | :-------------------------------: | :-------------: |
|                     40 / 40 / 20 / 100                     | 1. 256(v/b/f)-100(l)MiB / 0ms <br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms |              4               |                 1                 |       10        |
|                     40 / 40 / 20 / 100                     | 1. 256(v/b/f)-100(l)MiB / 0ms <br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms |              4               |                 2                 |       10        |
|                   100 / 100 / 50 / 1000                    | 1. 320(v/b/f)-100(l)MiB / 0ms<br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms  |              2               |                 1                 |       10        |
|                   100 / 100 / 50 / 1000                    | 1. 320(v/b/f)-100(l)MiB / 0ms<br />2. 320(v/b/f)-100(l)MiB / 100ms<br />3. 320(v/b/f)-100(i)MiB / 200ms  |              2               |                 2                 |       10        |

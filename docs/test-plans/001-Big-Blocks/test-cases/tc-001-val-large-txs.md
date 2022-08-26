# Test-Case #001 - Validators submit large txs

## Pre-Requisites:

1. Every validator has enough funds in the account
2. The chain has created the first block
3. Every validator has enough peers
4. Validators’ set is not changing during test execution
   1. All validators are created during genesis

## Steps for each of the validators:

1. Setups network with:
   1. `I` mb of bandwidth
   2. `J` milliseconds of network latency
2. Generates and broadcasts
   1. `X` kb of random data
   2. `Y` times
3. Checks the block size is bigger then 3.5 MiB

## Data Set:

| Number of Validators<br />`I` |            Bandwidth / Latency per validator<br />`J`             | KB of random data<br />`X` | Submit amount<br />`Y` |
| :---------------------------: | :---------------------------------------------------------------: | :------------------------: | :--------------------: |
|              20               | 1. 256MiB / 0ms <br /> 2. 256MiB / 100ms <br /> 3. 256MiB / 200ms |            200             |           10           |
|              40               |  1. 256MiB / 0ms <br />2. 320MiB / 100ms <br />3. 320MiB / 200ms  |            100             |           10           |
|              80               |  1. 320MiB / 0ms <br />2. 320MiB / 100ms <br />3. 320MiB / 200ms  |             50             |           10           |
|              100              |  1. 320MiB / 0ms <br />2. 320MiB / 100ms <br />3. 320MiB / 200ms  |             40             |           10           |

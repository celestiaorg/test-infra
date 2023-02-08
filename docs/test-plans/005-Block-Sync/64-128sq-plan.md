# Reconstruction for 32/64 Square Size Block

We need to run experiments to know how fast will it be for the full node(s) to sync blocks from bridge nodes and from amongst themselves, as the blocks are being produced.


In this scenario we are not having an assumption of withholding attacks

## Pre-Requisite
| Block Square Size<br />`I` |            Full Nodes amount <br />`J`             | 
| :---------------------------: | :-----------------------------------------: |
|              64               | 12 | 
|              128               |  12  | 

## HW Resources setup for Block Square Size ==  32

We already know that we can count on 172 cores CPU and 700 GB of RAM to allocate for testable nodes
Hence, our network setup/HW allocation will look like this:

1. 1 Validators with each having 1 CPU and 4 Gb
2. 3 Bridge Nodes with each having 1 CPU and 8 Gb
3. 12 Full Nodes with each having 1 CPU and 8 Gb

## Network Bandwidth:

- Validator/Bridge/Full: 256Mib -- 50ms

## Network Topology

### All cases
For all cases, the singular validator will be connected to all three bridges.

### Case 1 - Syncing Latest:

12 Full nodes all connected to different bridges from amongst the existing 3 bridges, and connected amongst themselves (_peerLimit is set to 3_) such that full nodes are syncing latest blocks as they are being produced.

### Case 2 - Syncing Latest with Hiccups:

12 Full nodes all connected to different bridges from amongst the existing 3 bridges, and connected amongst themselves (_peerLimit is set to 3_) such that full nodes are syncing latest blocks as they are being produced, with random full nodes being disconnecting from Bridge node a given random `hiccup-height`.

### Case 3 - Syncing Historical:

12 Full nodes all connected to different bridges from amongst the existing 3 bridges, and connected amongst themselves (_peerLimit is set to 3_) such that full nodes do not come alive until bridge nodes have synced up to `target-height` from core network, to produce historical syncing behavior

### Case 3 - Syncing Historical with Hiccups:

12 Full nodes all connected to different bridges from amongst the existing 3 bridges, and connected amongst themselves (_peerLimit is set to 3_) such that full nodes do not come alive until bridge nodes have synced up to `target-height` from core network, to produce historical syncing behavior, with random full nodes being disconnecting from Bridge node a given random `hiccup-height`.

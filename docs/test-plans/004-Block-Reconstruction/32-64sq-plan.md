# Reconstruction for 32/64 Square Size Block

We need to run experiments to know how long will it take for the full node(s) to reconstruct a block from a set of Light Nodes

In this scenario we are not having an assumption of withholding attacks

## Pre-Requisite
| Block Square Size<br />`I` |            Light Nodes amount <br />`J`             | 
| :---------------------------: | :-----------------------------------------: |
|              32               | 350 | 
|              64               |  1412  | 

## HW Resources setup for Block Square Size ==  32

We already know that we can count on 172 cores CPU and 700 GB of RAM to allocate for testable nodes
Hence, our network setup/HW allocation will look like this:

1. 5 Validators with each having 3 CPU and 4 Gb
2. 5 Bridge Nodes with each having 3 CPU and 4 Gb
3. 350 Light Nodes with each having 0.3 CPU and 1 Gb

### Full Node Setup/HW Allocation

- [ ] Case 1: 1 Full Node only having 37 CPU 256 Gb
- [ ] Case 2: 2 Full Nodes with each 16 CPU and 128 Gb
- [ ] Case 3: 3 Full Nodes with each 12 CPU and 64 Gb
- [ ] Case 4: 4 Full Nodes with each 8 CPU and 32 Gb

## Network Bandwidth:

- Validator/Bridge/Full: 320Mib -- 50ms
- Light: 100Mib -- 50ms

## Network Topology

### All Cases
For Cases 1-4 we enforce Full Nodes to blacklist all Bridge Nodes' multiaddresses to not receive any shares from them

### Case 2:
175 Light nodes can share to 1 Full Node
2 Full Nodes can communicate between one another as they are trusted peers to each other

### Case 3:
115 Light nodes can share to 1 Full Node
3 Full Nodes can communicate between one another as they are trusted peers to each other

### Case 4:
87 Light nodes can share to 1 Full Node
4 Full Nodes can communicate between one another as they are trusted peers to each other

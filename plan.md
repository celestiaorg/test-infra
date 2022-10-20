1. The Full Node has the latest head
2. All light nodes are network-bootstrapped and connected to the full node (no discovery required)
3. Share size is 32

To achieve 1), we need to:

A * Set a validator network of 5 performing PFDs to populate the DA layer
B * Let the bridge node reach a given height
C * Let the full node sync up to that given height
D * Spawn X amount of Light Nodes and have them DAS that given height (latest height)

A = like in 001-big-blocks/tests/app-sync/run_validator.go
    which is:
        - setup a validator network (genesis and connect everyone to everyone)
        - then start submitting pfds from each node
B = like in 001-big-blocks/tests/sync-past/run_bridge.go
C = like in 001-big-blocks/tests/sync-past/run_full.go
D = like in 001-big-blocks/tests/sync-past/run_light.go but TrustedHeader is pointing at the latest?

----

- create the validator network, create the genesis file, and do all the necessary shenanigans to init a new chain
- start submitting pdfs to the celestia-app
    - which the validators will include in a block
- create bridge nodes and let them start syncing up to a given height
- create full node and let them start syncing up to a given height
- create x light nodes and let them DAS the latest head and benchmark the time
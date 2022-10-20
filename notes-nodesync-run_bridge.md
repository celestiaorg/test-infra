



nodesync/run_bridge.go

1. retrieve execution-time from environment
2. define loger level
3. initialize the sync client (testground)
4. initialize network client (testground)
5. initialize the network configuration (testground)
    5.1 define latency and Bandwidth
    5.2 tell the config to update the state entry `network-configured` when done configuring
    5.3 configure routing policy to (allow all)
6. I don't understand `config.IPv4 = runenv.TestSubnet` but I can guess it's a way of retrieving an IP
7. More network shenanigans
8. Initialize the network
9. BuildBridge
    8.1 retrieve relevant validator info to build node by subscribing to AppNodeTopic (testground sync client) and getting the celestia app (validator underneath) info when created
    8.2 GetBlockHashByHeight (grpc call to the celestia app) for height 1
    8.3 Create celestia node with type bridge
    8.4 Start the node
    8.5 Retrieve block at height 2
    8.6 create a new subscription to publish bridge/s multiaddress
    8.7 Publish to `BridgeNodeTopic` the new bridge node's info

10. Sync up to given height
11. Record Message of what height did the node reach
12. A conditional to check if the node syncing
    12.1 If yes, record failure message
    12.2 If no, declare success but not declaring failure
13. stop and signal entry `testkit.FinishState`

So the bulk of the test is to test whether the DA bridge sync will happen within a given execution time
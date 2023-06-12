#!/bin/bash

# $1 is the first argument passed to the script (input number).
# $2 is the second argument passed to the script (string for if true).
input_number=$1
second_argument=$2

# Set the timeout and the uri as per your requirements
TIMEOUT=10
uri='http://endpoint_from_a_consensus_node.com'

# Execute the command before starting the loop.
celestia bridge start --p2p.network private

# Sleep for 10 seconds.
sleep 10

# For loop (replace with your own loop conditions).
for i in {1..10}
do
  # Fetch the data from the URL and extract the block height.
  block_height=$(wget --timeout=${TIMEOUT} -q -O - ${uri}:26657/block | jq -r '.result .block.height')

  # Compare the block_height to the input_number.
  if [ "$block_height" -eq "$input_number" ]; then
    # If the comparison is true, execute other commands here.
    echo "The block height equals the input number."

    # Store the bearer token into an environment variable.
    export CELESTIA_NODE_AUTH_TOKEN=$(celestia auth admit --p2p.network private)

    # Execute the second command with the second argument from the script.
    celestia rpc p2p BlockPeer $second_argument

  else
    echo "The block height does not equal the input number."
  fi

  # Sleep for 10 seconds.
  sleep 10
done

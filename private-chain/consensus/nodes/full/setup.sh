#!/bin/sh

# CELESTIA_HOME should be set in the environment
# CHAIN_ID should be set in the environment
# CONSENSUS_VALIDATOR_SERVICE should be set in the environment
# CONSENSUS_VALIDATOR_NAMEPACE should be set in the environment
# TIMEOUT optional

TIMEOUT="${TIMEOUT:-10}"

echo "---------------------------------------------------"
echo "Installing required tools..."
echo "---------------------------------------------------"

apk update --wait 10
apk add --wait 10 jq bind-tools

mkdir -p $CELESTIA_HOME/config/

echo "---------------------------------------------------"
echo "Getting consensus service IP"
echo "---------------------------------------------------"
CONSENSUS_SERVICE=$(dig ${CONSENSUS_VALIDATOR_SERVICE}.${CONSENSUS_VALIDATOR_NAMEPACE}.svc.cluster.local +short | uniq)

echo "---------------------------------------------------"
echo "Getting genesis.json file"
echo "---------------------------------------------------"
wget --timeout=${TIMEOUT} -q -O - ${CONSENSUS_VALIDATOR_SERVICE}.${CONSENSUS_VALIDATOR_NAMEPACE}.svc.cluster.local:26657/genesis? | jq -r '.result.genesis' > $CELESTIA_HOME/config/genesis.json

echo "---------------------------------------------------"
echo "Getting the celestia id"
echo "---------------------------------------------------"
CHAIN_ID=$(wget --timeout=${TIMEOUT} -q -O - ${CONSENSUS_VALIDATOR_SERVICE}.${CONSENSUS_VALIDATOR_NAMEPACE}.svc.cluster.local:26657/status? | jq -r '.result.node_info.network')

echo "---------------------------------------------------"
echo "Getting the Node id"
echo "---------------------------------------------------"
NODE_ID=$(wget --timeout=${TIMEOUT} -q -O - ${CONSENSUS_VALIDATOR_SERVICE}.${CONSENSUS_VALIDATOR_NAMEPACE}.svc.cluster.local:26657/status? | jq -r '.result.node_info.id')

PERSISTENT_PEERS=${NODE_ID}@${CONSENSUS_SERVICE}:26656
echo "---------------------------------------------------"
echo "Persistent Peers [${PERSISTENT_PEERS}]"
echo "---------------------------------------------------"

echo "---------------------------------------------------"
echo "Checking if the file [config.toml] exists or not..."
echo "---------------------------------------------------"
if [[ ! -f "$CELESTIA_HOME/config/config.toml" ]]; then
    touch $CELESTIA_HOME/config/config.toml
fi
if [[ -z "${CHAIN_ID}" ]]; then
    echo "Validator not ready yet.. (Could not fetch CHAIN_ID) Exiting..."
    exit 1
fi
if [[ -z "${NODE_ID}" ]]; then
    echo "Validator not ready yet.. (Could not fetch NODE_ID) Exiting..."
    exit 1
fi
if [[ ! -s "$CELESTIA_HOME/config/genesis.json" ]]; then
    echo "Genesis file doesn't exists yet... (Could not fetch genesis.json) Exiting..."
    exit 1
fi

echo "---------------------------------------------------"
echo "Current config.toml configuration =>"
cat $CELESTIA_HOME/config/config.toml
echo "---------------------------------------------------"

echo "---------------------------------------------------"
echo "Checking: base config"
echo "---------------------------------------------------"

# priv_validator_key_file
if grep -q "priv_validator_key_file" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^priv_validator_key_file *=.*/priv_validator_key_file = \"keys\/priv_validator_key.json\"/" $CELESTIA_HOME/config/config.toml
else
    echo priv_validator_key_file = \"keys\/priv_validator_key.json\" >> $CELESTIA_HOME/config/config.toml
fi
# node_key_file
if grep -q "node_key_file" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^node_key_file *=.*/node_key_file = \"keys/node_key.json\"/" $CELESTIA_HOME/config/config.toml
else
    echo node_key_file = \"keys/node_key.json\" >> $CELESTIA_HOME/config/config.toml
fi

echo "---------------------------------------------------"
echo "Checking: [persistent_peers]"
echo "---------------------------------------------------"
if grep -q "persistent_peers" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^persistent_peers *=.*/persistent_peers = \"${PERSISTENT_PEERS}\"/" $CELESTIA_HOME/config/config.toml
else
    echo "[p2p]" >> $CELESTIA_HOME/config/config.toml
    echo persistent_peers = \"${PERSISTENT_PEERS}\" >> $CELESTIA_HOME/config/config.toml
fi

echo "---------------------------------------------------"
echo "Checking: [prometheus]"
echo "---------------------------------------------------"
if grep -q "prometheus" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^prometheus *=.*/prometheus = true/" $CELESTIA_HOME/config/config.toml
else
    echo "[instrumentation]" >> $CELESTIA_HOME/config/config.toml
    echo "prometheus = true" >> $CELESTIA_HOME/config/config.toml
fi

echo "---------------------------------------------------"
echo "Checking: [tx_index]"
echo "---------------------------------------------------"
if grep -q "tx_index" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^indexer *=.*/indexer = \"kv\"/" $CELESTIA_HOME/config/config.toml
else
    echo "[tx_index]" >> $CELESTIA_HOME/config/config.toml
    echo "indexer = \"kv\"" >> $CELESTIA_HOME/config/config.toml
fi

echo "---------------------------------------------------"
echo "Configuration applied =>"
cat $CELESTIA_HOME/config/config.toml
echo "---------------------------------------------------"

echo "---------------------------------------------------"
echo "Tweaking the config - block reconstruction"
echo "---------------------------------------------------"
sed -i 's/max_subscription_clients = 100/max_subscription_clients = 6000/g' /home/celestia/config/config.toml
sed -i 's/max_subscriptions_per_client = 5/max_subscriptions_per_client = 1000/g' /home/celestia/config/config.toml
echo "---------------------------------------------------"
cat /home/celestia/config/genesis.json
echo "---------------------------------------------------"

#!/bin/sh
# CELESTIA_HOME should be set in the environment
# APP_ID should be set in the environment
# CHAIN_ID should be set in the environment
# INITIAL_TIA_AMOUNT should be set in the environment
# STAKING_TIA_AMOUNT should be set in the environment

# EVM_ADDRESS should be set in the environment

KEY_NAME="${APP_ID}-${CHAIN_ID}"

echo "---------------------------------------------------"
echo "Checking if the file [config.toml] exists or not..."
echo "---------------------------------------------------"
mkdir -p $CELESTIA_HOME/config/
if [[ ! -f "$CELESTIA_HOME/config/config.toml" ]]; then
    touch $CELESTIA_HOME/config/config.toml
fi

if [[ ! -f "$CELESTIA_HOME/config/genesis.json" ]]; then

    celestia-appd init "${APP_ID}" --home "${CELESTIA_HOME}" --chain-id "${CHAIN_ID}"

   # yes n | celestia-appd keys add "${KEY_NAME}" --home "${CELESTIA_HOME}" --no-backup
    celestia-appd keys add "${KEY_NAME}" --home "${CELESTIA_HOME}" --no-backup

    account_address=$(celestia-appd keys show "${KEY_NAME}" -a --home "${CELESTIA_HOME}")
    celestia-appd add-genesis-account "${account_address}" "${INITIAL_TIA_AMOUNT}" --home "${CELESTIA_HOME}"

    celestia-appd gentx "${KEY_NAME}" "${STAKING_TIA_AMOUNT}" --home "${CELESTIA_HOME}" --chain-id "${CHAIN_ID}" --evm-address "${EVM_ADDRESS}"

    celestia-appd collect-gentxs --home "${CELESTIA_HOME}"

    echo "Copying the keys to the config path..."
    mv $CELESTIA_HOME/config/priv_validator_key.json $CELESTIA_HOME/keys
    mv $CELESTIA_HOME/config/node_key.json $CELESTIA_HOME/keys
fi

echo "---------------------------------------------------"
echo "Checking: base config"
echo "---------------------------------------------------"

# priv_validator_key_file
if grep -q "priv_validator_key_file" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^priv_validator_key_file *=.*/priv_validator_key_file = \"keys\/priv_validator_key.json\"/" $CELESTIA_HOME/config/config.toml
else
    echo priv_validator_key_file = \"keys/priv_validator_key.json\" >> $CELESTIA_HOME/config/config.toml
fi
# node_key_file
if grep -q "node_key_file" "${CELESTIA_HOME}/config/config.toml"; then
    sed -i.bak -e "s/^node_key_file *=.*/node_key_file = \"keys\/node_key.json\"/" $CELESTIA_HOME/config/config.toml
else
    echo node_key_file = \"keys/node_key.json\" >> $CELESTIA_HOME/config/config.toml
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
echo "Tweaking the config - block reconstruction"
echo "---------------------------------------------------"
sed -i 's/"gov_max_square_size": "64"/"gov_max_square_size": "128"/g' /home/celestia/config/genesis.json
sed -i 's/max_subscription_clients = 100/max_subscription_clients = 6000/g' /home/celestia/config/config.toml
sed -i 's/max_subscriptions_per_client = 5/max_subscriptions_per_client = 1000/g' /home/celestia/config/config.toml

sed -i 's/"max_bytes": "1974272"/"max_bytes": "8388608"/g' /home/celestia/config/genesis.json
sed -i 's/"max_deposit_period": "172800s"/"max_deposit_period": "20s"/g' /home/celestia/config/genesis.json
sed -i 's/"voting_period": "172800s"/"voting_period": "20s"/g' /home/celestia/config/genesis.json
echo "---------------------------------------------------"
cat /home/celestia/config/genesis.json
echo "---------------------------------------------------"

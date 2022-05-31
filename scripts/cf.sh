#!/bin/sh


CHAINID="tia-test"

#need to adjust the amount of validators
coins="1000000000000000utia"
celestia-appd init xm1 --chain-id tia-test --home ~/.celestia-app-1 
celestia-appd keys add xm1 --keyring-backend="test" --home ~/.celestia-app-1 
celestia-appd keys add xm2 --keyring-backend="test" --home ~/.celestia-app-2 
celestia-appd keys add xm3 --keyring-backend="test" --home ~/.celestia-app-3 
celestia-appd keys add xm4 --keyring-backend="test" --home ~/.celestia-app-4 
celestia-appd keys add xm5 --keyring-backend="test" --home ~/.celestia-app-5 
celestia-appd keys add xm6 --keyring-backend="test" --home ~/.celestia-app-6 
celestia-appd keys add xm7 --keyring-backend="test" --home ~/.celestia-app-7 
celestia-appd keys add xm8 --keyring-backend="test" --home ~/.celestia-app-8 
celestia-appd keys add xm9 --keyring-backend="test" --home ~/.celestia-app-9 
celestia-appd keys add xm10 --keyring-backend="test" --home ~/.celestia-app-10
#need to find a way to scrape the output of each to continue below automatically

#change addresses accordingly
celestia-appd add-genesis-account celestia1mld039ypx3wu82h9wua4vjygze7es3s6rl9xfl 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1exxn5t7hzrj7dm0mexxdnyyg059uwu7qt9uelk 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia153k7x84r0x6uh843lku26nyl57t5twe9fcu482 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1j5zz88ptwdxmngec7q6dmvt9rzhranlqqecnw2 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1x3vr5ft6pzrks2j60y3us2hgq3jyrz4zscusa4 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1caj0ltdh22czdancskhnk4jk048803s69vpwcf 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1nrj4pnnvn9jasgdr222ul5ycqhd6us6avl492z 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1vnxxkfxnv38zjc08jp44agrjpf7d2a8za5959k 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia13udnyn4nw58r06fx3xjwk90vt983mareejztrh 1000000000000000utia --home ~/.celestia-app-1
celestia-appd add-genesis-account celestia1m0rl7fdmwtaht0kkrh8eetynvrumaa52l7zphm 1000000000000000utia --home ~/.celestia-app-1


cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-2/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-3/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-4/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-5/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-6/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-7/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-8/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-9/config/genesis.json 
cp ~/.celestia-app-1/config/genesis.json ~/.celestia-app-10/config/genesis.json 


celestia-appd gentx xm1 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-1
celestia-appd gentx xm2 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-2
celestia-appd gentx xm3 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-3
celestia-appd gentx xm4 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-4
celestia-appd gentx xm5 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-5
celestia-appd gentx xm6 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-6
celestia-appd gentx xm7 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-7
celestia-appd gentx xm8 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-8
celestia-appd gentx xm9 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-9
celestia-appd gentx xm10 5000000000utia --keyring-backend="test" --chain-id tia-test --home ~/.celestia-app-10

#renaming step is needed to not bother with automatic generation of file names 
cp /Users/bidon4/.celestia-app-1/config/gentx/gentx-34eeb37dc4c109a86295515620d4c0f54683b8ef.json
cp /Users/bidon4/.celestia-app-2/config/gentx/gentx-d2187f3eea2e4abff74fb7346f5ea0ddacff7288.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-3/config/gentx/gentx-47dcbdf85eee4218a5bdabc7d63959664b6e570e.json .celestia-app-1/config/gentx 
cp /Users/bidon4/.celestia-app-4/config/gentx/gentx-fa09ce78e38d9d0fd423e41b136101a5155818c8.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-5/config/gentx/gentx-26c9a68e138a3d2e191769b576d4de62d98f6389.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-6/config/gentx/gentx-430d35c6eba215b7cf90d1575a931a3569ecfc2c.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-7/config/gentx/gentx-3cc9a98b2f9070b6f25220c9987d1afd67867603.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-8/config/gentx/gentx-4774e427a71c8c93fd4d7d806552fe9440ee8253.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-9/config/gentx/gentx-d8b1e2041e47d89f8f7b2d29f552ba93e64c00b1.json .celestia-app-1/config/gentx
cp /Users/bidon4/.celestia-app-10/config/gentx/gentx-bff33dffaf94b1e7b1e002c847e46bc9494876d4.json .celestia-app-1/config/gentx

celestia-appd collect-gentxs --home ~/.celestia-app-1 

# repeat cp genesis afterwards
# sed of viper the config.toml file with full to validator and timeout-commit values

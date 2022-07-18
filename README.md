# test-infra
Testing infrastructure for the Celestia Network

Please install testground before executing the test-plan

After installing, follow these commands

```bash
cd test-infra
testground plan --import . --name celestia

# This command should be executed in the 1st terminal 
testground daemon

# This command should be executed in the 2nd terminal
testground run composition -f local-compositions/gen-validators.toml --wait
```
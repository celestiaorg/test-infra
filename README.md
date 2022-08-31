# Testing & Infrastructure :microscope: :globe_with_meridians:

Testing scenarios and network infrastructure for the Celestia Network

## Pre-Requisites

Please install `docker` and [testground](https://docs.testground.ai/v/master/getting-started)
to execute network tests.

## Go requirements

| Requirement | Notes          |
| ----------- | -------------- |
| Go version  | 1.18 or higher |

## System Requirements

We have compositions that are separated into 2 environments:

1. `local:docker`
2. `cluster:k8s`

| Environment  | CPU (cores) | RAM (Gib) |
| ------------ | :---------: | :-------: |
| local:docker |    8~16     |   16~32   |
| cluster:k8s  |  3000~4000  | 4000~5000 |

At the moment, we are only using `docker:generic` as a builder.
Please, check our `Dockerfile` for more information.

## Repo Navigation

The repository is divided into 4 main directories:

1. `docs`
2. `compositions`
3. `tests`
4. `testkit`

The order of directories above :point_up: is how the repo should be read
if you want to get acquinted with test plans/cases design and their further implementations.
Each of the directories contains its own `README.md`

## Test Execution

```bash
cd test-infra
testground plan --import . --name celestia

# This command should be executed in the 1st terminal
testground daemon

# This command should be executed in the 2nd terminal
testground run composition -f compositions/local-docker/001-val-large-txs-3.toml --wait
```

## Code of Conduct

See our Code of Conduct [here](https://docs.celestia.org/community/coc).

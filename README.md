# Testing & Infrastructure :microscope: :globe_with_meridians:

Testing scenarios and network infrastructure for the Celestia Network

## Pre-Requisites

Please install `docker` and [testground](https://docs.testground.ai/v/master/getting-started) to execute network tests.

To install testground, please perform the following:
```bash
$ make install-tg
```

## Go requirements

| Requirement | Notes          |
| ----------- | -------------- |
| Go version  | 1.19 or higher |

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
2. `plans`
4. `testkit`

The order of directories above :point_up: is how the repo should be read
if you want to get acquainted with test plans/cases design and their further implementations.
Each of the directories contains its own `README.md`.

For test plans, each test plan resides in its own directory under `plans/TP_NAME` and acts as separate golang module with its own `manifest.toml` and `Dockerfile`.

### Testkit caveats 
Note that `testkit` is shared between all test plans, and acts a separate golang module. At the moment, all test plans are using the following testkit version:
```
github.com/celestiaorg/test-infra/testkit v0.0.0-20221020113323-2f2873f97406
```

if you make any changes to `testkit` make sure to commit and retrieve the commit hash, and then retrieve the go module version by running:
```bash
$ go list -m github.com/celestiaorg/test-infra/testkit@commithash
```
Then use the output to replace the existing version in `plans/YOUR_PLAN/go.mod` in the line where `testkit` is required as a a dependency.

## Create a new testplan

Change the current working directory by `cd`-ing into your clone of this repository and then temporarily set `$TESTGROUND_HOME` to point to it, and then create your plan by running the following command:
```bash
$ make tg-create-testplan NAME=YOUR_DESIRED_TESTPLAN_NAME
```

This will create a new go module with a `manifest.toml` under `./plans/YOUR_DESIRED_TESTPLAN_NAME` and a documentation folder  and file under `./docs/test-plans/YOUR_DESIRED_TESTPLAN_NAME/` to get you up to speed.

> **Important**: Make sure to update the created `go.mod` under your test plan's new directory as it comes with default dummy values from testground. Namely, udpate the golang version as it comes with `go 1.14` by default and you module's name as it comes with `github.com/your/module/name`

## Test Execution

1. Import your desired test plan into `TESTGROUND_HOME`
```bash
$ cd test-infra
$ make tg-import-testplan NAME=001-big-blocks TESTPLAN=001-big-blocks
```
Available plans are in `./plans`

2. Launch the testground daemon
```bash
$ testground daemon
```

3. In another terminal, run a composition of your testplan
```
$ make tg-run-composition TESTPLAN=YOUR_TEST_PLAN RUNNER=DESIRED_RUNNER COMPOSITION=COMPOSITION_NAME
```
Note: `COMPOSITION` should only include the composition's filename, without the `.toml` extension
`RUNNER` by default is `local-docker`. Another possible value is `cluster-k8s`

Example:
```
$ make tg-run-composition RUNNER=local-docker TESTPLAN=001-big-blocks COMPOSITION=002-da-sync-12
```

## Code of Conduct

See our Code of Conduct [here](https://docs.celestia.org/community/coc).

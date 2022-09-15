# Compositions

Please navigate to the `manifest.toml` if you want to know more about which test cases/params are defined for the compositions to set during test runs

## local-docker

This directory contains sanity compositions that can be easily run on a local PC or any small VM(e.g. DO droplet). The motivation is to do quick regression check-ups if any PR arises from the stack(core/app/node etc).

## cluster-k8s

This directory contains compositions that are described in `docs/test-plans`. The sorting of inner directories are following the same pattern as the test-plan to test-case placement. Namings of directories and files follow this style:

`test-case-id` -> `participants-amount` -> `bandwidth-latency-per-participant`

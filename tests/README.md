# Tests

Each `id-*.go` follows the naming of the test-case in `docs/test-plan/id-test-plan/*`. 

In order to not clutter the test-file with different participants' steps, each dedicated directory is created to 
maintain steps of each types of participants. 

## Helpers

Helpers contain common scenarios per each of node type(e.g. funding/sync current/sync past headers)
The `common` directory maintains steps for a participant that plays the same role in every of the test-case (e.g. creating a validators set).

## Plans

In plans directory you will find all test-cases per each of a test-plan that is described in docs

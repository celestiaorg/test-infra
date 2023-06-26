/*
Package qgbKit is a wrapper around the QGB orchestrator-relayer implementation.

In order to ease the calling of commands like the end user does through CLI,
this package is providing easy to use functions for more readability for the test
scenario.

To start using the wrapped style CLI, you need to initialise the struct QGBKit.
A returned cmd contains functions that returns an output from StdOut pipeline
(e.g. like the end user will see in the terminal) as well as errors if something
bad happened while executing a command

wrappedCmd := qgbkit.New()

Currently, the way it is used is via creating a validator then attaching an orchestrator to it as a
side car. This is similar to how we expect validators to run orchestrators. However, we can opt for different
design where we spin up a Celestia network. Then, the we orchestrators start via taking the IP address of a
validator along with the private key to its EVM address.
*/
package qgbkit

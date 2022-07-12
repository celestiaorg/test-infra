/*
Package appkit is a wrapper around the App Client and Cosmos's Server cmds

In order to ease the calling of commands like the end user does through CLI,
this package is providing easy to use functions for more readability for the test
scenario.

To start using the wrapped style CLI, you need to initialise the struct AppKit.
A returned cmd contains functions that returns an output from StdOut pipeline
(e.g. like the end user will see in the terminal) as well as errors if something
bad happened while executing a command

Other functionality in appkit is an easy-to-modify values in .toml(e.g. config.toml)
This can help the test user to modify what is needed for a scenario without a
boilerplate code from viper

Last but not least is the REST API calls. Appkit takes care of all boilerplate code
for doing simple http requests

wrappedCmd := appkit.New()
output, err := wrappedCmd.InitChain("moniker", "test-chain", "/path/to/store")
err = appkit.ChangeNodeMode("/path/to/config.toml", "seed")
hash, err = appkit.GetBlockByHeight(net.Parse("127.0.0.1"), 10)
*/
package appkit

// Welcome, testground plan writer!
// If you are seeing this for the first time, check out our documentation!
// https://app.gitbook.com/@protocol-labs/s/testground/

package main

import "github.com/ipfs/testground/sdk/runtime"

func main() {
	runtime.Invoke(run)
}

func run(runenv *runtime.RunEnv) error {
	runenv.RecordMessage("Hello, Testground!")
	return nil
}

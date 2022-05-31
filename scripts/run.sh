#!/bin/sh

testground run single --plan celestia-app --testcase capp-1 --builder docker:generic --runner local:docker --wait --instances 10
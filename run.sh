#!/bin/bash
# This script is support restart for testctl

go install
go install ../testing-tools
go install ./testnet

# Only delete test log
rm -rf test
rm -rf test/storage-*/logs/

# If need restore previous data from disk,should add "--restart"
testing-tools
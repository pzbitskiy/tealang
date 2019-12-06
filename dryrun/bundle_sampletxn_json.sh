#!/usr/bin/env bash

THISDIR=$(dirname $0)

cat <<EOM | gofmt > $THISDIR/sampletxn_gen.go
// Code generated during build process. DO NOT EDIT.
package dryrun

var sampleTxnData []byte

type txnDesc struct {
	Sender 			string		// Algorand Address, base32-encoded string with checksum
	Fee 			uint64
	FirstValid 		uint64
	LastValid 		uint64
	Note 			string		// base64-encoded
	Lease 			string		// base64-encoded
	Receiver 		string		// Algorand Address, base32-encoded string with checksum
	Amount 			uint64
	CloseRemainderTo string		// Algorand Address, base32-encoded string with checksum
	VotePK 			string		// base64-encoded
	SelectionPK 	string		// base64-encoded
	VoteFirst 		uint64
	VoteLast 		uint64
	VoteKeyDilution uint64
	Type 			string		// string
	TypeEnum 		uint64
	XferAsset 		uint64
	AssetAmount 	uint64
	AssetSender 	string		// Algorand Address, base32-encoded string with checksum
	AssetReceiver 	string		// Algorand Address, base32-encoded string with checksum
	AssetCloseTo 	string		// Algorand Address, base32-encoded string with checksum
	GroupIndex 		uint64
}

func init() {
	sampleTxnData = []byte{
        $(cat $THISDIR/sampletxn.json | hexdump -v -e '1/1 "0x%02X, "' | fmt)
	}
}

EOM
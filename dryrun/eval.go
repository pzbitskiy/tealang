package dryrun

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/algorand/go-algorand/config"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/transactions"
	"github.com/algorand/go-algorand/data/transactions/logic"
	"github.com/algorand/go-algorand/protocol"
)

//go:generate sh ./bundle_sampletxn_json.sh

// Run bytecode using transaction data from txnFile file
func Run(bytecode []byte, txnFile string, trace io.Writer) (bool, error) {
	txn, err := loadTxn(txnFile)
	if err != nil {
		return false, err
	}

	stxn := transactions.SignedTxn{}
	stxn.Txn = txn
	proto := config.Consensus[protocol.ConsensusCurrentVersion]

	ep := logic.EvalParams{Txn: &stxn, Proto: &proto}
	err = logic.Check(bytecode, ep)
	if err != nil {
		return false, err
	}

	txgroup := make([]transactions.SignedTxn, 1)
	txgroup[0] = stxn

	ep = logic.EvalParams{
		Txn:        &stxn,
		Proto:      &proto,
		Trace:      trace,
		TxnGroup:   txgroup,
		GroupIndex: 1,
	}

	pass, err := logic.Eval(bytecode, ep)
	return pass, err
}

func loadTxn(txnFile string) (txn transactions.Transaction, err error) {
	var txnData []byte
	if txnFile != "" {
		txnData, err = ioutil.ReadFile(txnFile)
		if err != nil {
			return
		}
	} else {
		txnData = sampleTxnData
	}

	var sampleTxn txnDesc
	err = json.Unmarshal(txnData, &sampleTxn)
	if err != nil {
		return
	}

	txn.Type = protocol.TxType(sampleTxn.Type)
	if txn.Sender, err = basics.UnmarshalChecksumAddress(sampleTxn.Sender); err != nil {
		return
	}
	txn.Fee = basics.MicroAlgos{sampleTxn.Fee}
	txn.FirstValid = basics.Round(sampleTxn.FirstValid)
	txn.LastValid = basics.Round(sampleTxn.LastValid)
	if txn.Note, err = base64.StdEncoding.DecodeString(sampleTxn.Note); err != nil {
		return
	}
	var lease []byte
	if lease, err = base64.StdEncoding.DecodeString(sampleTxn.Lease); err != nil {
		return
	}
	copy(txn.Lease[:], lease)

	txn.Amount = basics.MicroAlgos{sampleTxn.Amount}
	if txn.Receiver, err = basics.UnmarshalChecksumAddress(sampleTxn.Receiver); err != nil {
		return
	}
	if txn.CloseRemainderTo, err = basics.UnmarshalChecksumAddress(sampleTxn.CloseRemainderTo); err != nil {
		return
	}

	txn.XferAsset = basics.AssetIndex(sampleTxn.XferAsset)
	txn.AssetAmount = sampleTxn.AssetAmount
	if txn.AssetSender, err = basics.UnmarshalChecksumAddress(sampleTxn.AssetSender); err != nil {
		return
	}
	if txn.AssetReceiver, err = basics.UnmarshalChecksumAddress(sampleTxn.AssetReceiver); err != nil {
		return
	}
	if txn.AssetCloseTo, err = basics.UnmarshalChecksumAddress(sampleTxn.AssetCloseTo); err != nil {
		return
	}

	return txn, nil
}

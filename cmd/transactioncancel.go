// Copyright © 2017-2019 Weald Technology Trading
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wealdtech/ethereal/v2/cli"
	string2eth "github.com/wealdtech/go-string2eth"
)

var transactionCancelAmount string
var transactionCancelToAddress string

// transactionCancelCmd represents the transaction up command
var transactionCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a pending transaction",
	Long: `Cancel a pending transaction.  For example:

    ethereal transaction cancel --transaction=0x454d2274155cce506359de6358785ce5366f6c13e825263674c272eec8532c0c

Note that Ethereum does not have the ability to cancel a pending transaction, so this overwrites the pending transaction with a 0-value transfer back to the address sender.  It will, however, still need to be mined so choose an appropriate gas price.  If not supplied then the gas price will default to just over 10% higher than the gas price of the transaction to be cancelled.

The cancellation transaction will cost 21000 gas.

This will return an exit status of 0 if the transaction is successfully submitted (and mined if --wait is supplied), 1 if the transaction is not successfully submitted, and 2 if the transaction is successfully submitted but not mined within the supplied time limit.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli.Assert(transactionStr != "", quiet, "--transaction is required")
		txHash := common.HexToHash(transactionStr)
		ctx, cancel := localContext()
		defer cancel()
		tx, pending, err := client.TransactionByHash(ctx, txHash)
		cli.ErrCheck(err, quiet, fmt.Sprintf("Failed to obtain transaction %s", txHash.Hex()))
		cli.Assert(pending, quiet, fmt.Sprintf("Transaction %s has already been mined", txHash.Hex()))

		minGasPrice := new(big.Int).Add(new(big.Int).Add(tx.GasPrice(), new(big.Int).Div(tx.GasPrice(), big.NewInt(10))), big.NewInt(1))
		if viper.GetString("gasprice") == "" {
			// No gas price supplied; use the calculated minimum
			gasPrice = minGasPrice
		} else {
			// Gas price supplied; ensure it is over 10% more than the current gas price
			cli.Assert(gasPrice.Cmp(minGasPrice) > 0, quiet, fmt.Sprintf("Gas price must be at least %s", string2eth.WeiToString(minGasPrice, true)))
		}

		// Create and sign the transaction
		fromAddress, err := txFrom(tx)
		cli.ErrCheck(err, quiet, "Failed to obtain from address")

		nonce = int64(tx.Nonce())
		signedTx, err := createSignedTransaction(fromAddress, &fromAddress, nil, gasLimit, nil)
		cli.ErrCheck(err, quiet, "Failed to create transaction")

		if offline {
			if !quiet {
				buf := new(bytes.Buffer)
				cli.ErrCheck(signedTx.EncodeRLP(buf), quiet, "failed to encode transaction")
				fmt.Printf("0x%s\n", hex.EncodeToString(buf.Bytes()))
			}
			os.Exit(exitSuccess)
		}

		ctx, cancel = localContext()
		defer cancel()
		err = client.SendTransaction(ctx, signedTx)
		cli.ErrCheck(err, quiet, "Failed to send transaction")
		handleSubmittedTransaction(signedTx, log.Fields{
			"group":            "transaction",
			"command":          "cancel",
			"oldtransactionid": txHash.Hex(),
		}, true)
	},
}

func init() {
	transactionCmd.AddCommand(transactionCancelCmd)
	transactionFlags(transactionCancelCmd)
	transactionCancelCmd.Flags().StringVar(&transactionCancelAmount, "amount", "", "Amount of Ether to transfer")
	transactionCancelCmd.Flags().StringVar(&transactionCancelToAddress, "to", "", "Address to which to transfer Ether")
	addTransactionFlags(transactionCancelCmd, "the address that holds the funds")
}

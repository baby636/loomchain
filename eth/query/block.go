// +build evm

package query

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	searchBlockSize = uint64(100)
)

func GetBlockByNumber(state loomchain.ReadOnlyState, height uint64, full bool, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
	params := map[string]interface{}{}
	params["heightPtr"] = &height
	var blockresult *ctypes.ResultBlock
	iHeight := int64(height)
	blockresult, err := core.Block(&iHeight)
	if err != nil {
		return nil, err
	}
	blockinfo := types.EthBlockInfo{
		Hash:       blockresult.BlockMeta.BlockID.Hash,
		ParentHash: blockresult.Block.Header.LastBlockID.Hash,

		Timestamp: int64(blockresult.Block.Header.Time.Unix()),
	}
	if uint64(state.Block().Height) == height {
		blockinfo.Number = 0
	} else {
		blockinfo.Number = int64(height)
	}

	txHash, err := readReceipts.GetTxHash(state, height)
	if err != nil {
		return nil, errors.Wrap(err, "getting tx hash")
	}
	if len(txHash) > 0 {
		bloomFilter, err := readReceipts.GetBloomFilter(state, height)
		if err != nil {
			return nil, errors.Wrap(err, "reading bloom filter")
		}
		blockinfo.LogsBloom = bloomFilter
		if full {
			txReceipt, err := readReceipts.GetReceipt(state, txHash)
			if err != nil {
				return nil, errors.Wrap(err, "reading receipt")
			}
			txReceiptProto, err := proto.Marshal(&txReceipt)
			if err != nil {
				return nil, errors.Wrap(err, "marshall receipt")
			}
			blockinfo.Transactions = append(blockinfo.Transactions, txReceiptProto)
		} else {
			blockinfo.Transactions = append(blockinfo.Transactions, txHash)
		}
	}

	return proto.Marshal(&blockinfo)
}

func GetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
	start := uint64(state.Block().Height)
	var end uint64
	if uint64(start) > searchBlockSize {
		end = uint64(start) - searchBlockSize
	} else {
		end = 1
	}

	for start > 0 {
		var info *ctypes.ResultBlockchainInfo
		info, err := core.BlockchainInfo(int64(end), int64(start))
		if err != nil {
			return nil, err
		}

		if err != nil {
			return nil, err
		}
		for i := int(len(info.BlockMetas) - 1); i >= 0; i-- {
			if 0 == bytes.Compare(hash, info.BlockMetas[i].BlockID.Hash) {
				return GetBlockByNumber(state, uint64(int(end)+i), full, readReceipts)
			}
		}

		if end == 1 {
			return nil, fmt.Errorf("can't find block to match hash")
		}

		start = end
		if uint64(start) > searchBlockSize {
			end = uint64(start) - searchBlockSize
		} else {
			end = 1
		}
	}
	return nil, fmt.Errorf("can't find block to match hash")
}

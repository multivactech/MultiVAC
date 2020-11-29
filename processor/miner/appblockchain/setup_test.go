package appblockchain

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"

	"github.com/multivactech/MultiVAC/base/rlp"
	"github.com/multivactech/MultiVAC/interface/iblockchain"
	"github.com/multivactech/MultiVAC/interface/idepositpool"
	"github.com/multivactech/MultiVAC/interface/isysapi"
	"github.com/multivactech/MultiVAC/model/chaincfg/genesis"
	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
	"github.com/multivactech/MultiVAC/model/chaincfg/signature"
	"github.com/multivactech/MultiVAC/model/merkle"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
)

// txGen used to generate a fake tx only used for tsting
type txGen struct {
	outPool []*wire.OutPoint
	pkPool  *genesis.Pool
}

func newTxGen(shard shard.Index) *txGen {
	tg := &txGen{
		outPool: make([]*wire.OutPoint, 0),
		pkPool:  genesis.NewPool(),
	}
	genesisOuts := genesis.GenesisBlocks[shard].Body.Outs
	for _, out := range genesisOuts {
		tg.outPool = append(tg.outPool, &out.OutPoint)
	}
	return tg
}

// Pramas outType specify the type of txIn.PreviousOutPoint
// Note: only support deposit out and transfer out
func (tg *txGen) getRawTx(outType wire.OutType, shard shard.Index) (*wire.MsgTx, error) {
	for i, out := range tg.outPool {
		if out.Type == outType {
			tx := wire.NewMsgTx(wire.TxVersion, shard)
			tx.TxIn = append(tx.TxIn, &wire.TxIn{
				PreviousOutPoint: *out,
			})
			tg.outPool = append(tg.outPool[0:i], tg.outPool[i+1:]...)
			return tx, nil
		}
	}
	return nil, fmt.Errorf("No avaliable tx")
}

func (tg *txGen) makeDepositTx(shard shard.Index, to, binding multivacaddress.Address, value *big.Int) (*wire.MsgTx, error) {
	rawTx, err := tg.getRawTx(wire.Transfer, shard)
	if err != nil {
		return nil, err
	}
	rawTx.API = isysapi.SysAPIDeposit
	p := isysapi.DepositParams{
		TransferParams: &isysapi.TransferParams{
			To:     to,
			Shard:  shard,
			Amount: value,
		},
		BindingAddress: binding,
	}
	d, err := rlp.EncodeToBytes(p)
	if err != nil {
		return nil, err
	}
	rawTx.Params = d
	err = tg.sign(rawTx)
	if err != nil {
		return nil, err
	}
	return rawTx, nil
}

func (tg *txGen) makeTransferTx(shard shard.Index, to multivacaddress.Address, value *big.Int) (*wire.MsgTx, error) {
	rawTx, err := tg.getRawTx(wire.Transfer, shard)
	if err != nil {
		return nil, err
	}
	rawTx.API = isysapi.SysAPITransfer
	p := isysapi.TransferParams{
		To:     to,
		Shard:  shard,
		Amount: value,
	}
	d, err := rlp.EncodeToBytes(p)
	if err != nil {
		return nil, err
	}
	rawTx.Params = d
	err = tg.sign(rawTx)
	if err != nil {
		return nil, err
	}
	return rawTx, nil
}

func (tg *txGen) sign(tx *wire.MsgTx) error {
	out := tx.TxIn[0].PreviousOutPoint
	sk, err := tg.pkPool.GetSkFromAddress(out.UserAddress)
	if err != nil {
		return err
	}
	skHash := signature.PrivateKey(sk)
	tx.Sign(&skHash)
	if err := tx.VerifyTransaction(); err != nil {
		return err
	}
	return nil
}

type toolsConfig struct {
	to      multivacaddress.Address
	binding multivacaddress.Address
	value   *big.Int
}

type testTools struct {
	config     *toolsConfig
	tree       *merkle.FullMerkleTree
	ledgerInfo wire.LedgerInfo
	shard      shard.Index
	height     int64
	blockchain iblockchain.BlockChain
	dPool      idepositpool.DepositPool
	abci       *appBlockChain
	txGen      *txGen
	t          *testing.T

	txGenType string
}

func newTools(blockchain iblockchain.BlockChain, abci *appBlockChain, dPool idepositpool.DepositPool, txGen *txGen, t *testing.T, cfg *toolsConfig) *testTools {
	tools := &testTools{
		tree:       merkle.NewFullMerkleTree(),
		blockchain: blockchain,
		dPool:      dPool,
		abci:       abci,
		txGen:      txGen,
		t:          t,
		shard:      abci.shardIndex,
		txGenType:  "transfer",
		config:     cfg,
	}

	// Load genesis block
	genesisBlock := genesis.GenesisBlocks[abci.shardIndex]
	outsMerkleTree := genesis.GenesisBlocksOutsMerkleTrees[tools.shard]

	tools.height = 1
	err := tools.tree.Append(&genesisBlock.Header.OutsMerkleRoot)
	if err != nil {
		tools.t.Error(err)
	}
	err = tools.tree.HookSecondLayerTree(outsMerkleTree)
	if err != nil {
		tools.t.Error(err)
	}

	tools.ledgerInfo.SetShardHeight(tools.shard, 1)
	tools.ledgerInfo.Size = 1
	tools.blockchain.ReceiveBlock(genesisBlock)

	return tools
}

func (tools *testTools) setTxGenType(txType string) {
	tools.txGenType = txType
}

func (tools *testTools) receiveBlock(block *wire.MsgBlock) error {
	if block.Header.Height <= 1 {
		return fmt.Errorf("invalid block height")
	}

	tools.blockchain.ReceiveBlock(block)

	if block.Header.IsEmptyBlock {
		fmt.Println("tools receive an empty block")
		return nil
	}

	for _, tx := range block.Body.Transactions {
		if tx.Tx.IsReduceTx() {
			continue
		}
		for _, txIn := range tx.Tx.TxIn {
			previousOutState := wire.OutState{
				OutPoint: txIn.PreviousOutPoint,
				State:    wire.StateUnused,
			}
			previousHash := merkle.MerkleHash(sha256.Sum256(previousOutState.ToBytesArray()))
			newOutState := wire.OutState{
				OutPoint: txIn.PreviousOutPoint,
				State:    wire.StateUsed,
			}
			newHash := merkle.MerkleHash(sha256.Sum256(newOutState.ToBytesArray()))
			err := tools.tree.UpdateLeaf(&previousHash, &newHash)
			if err != nil {
				return err
			}
		}
	}

	ledgerInfo := block.Body.LedgerInfo
	for _, shardIndex := range shard.ShardList {
		newHeight := ledgerInfo.GetShardHeight(shardIndex)
		preHeight := tools.ledgerInfo.GetShardHeight(shardIndex)
		for height := preHeight + 1; height <= newHeight; height++ {
			b := tools.blockchain.GetShardsBlockByHeight(shardIndex, wire.BlockHeight(height))
			err := tools.tree.Append(&b.Header.OutsMerkleRoot)
			if err != nil {
				return err
			}
			leaves := make([]*merkle.MerkleHash, 0)
			for i := 0; i < len(b.Body.Outs); i++ {
				leafHash := merkle.ComputeMerkleHash(b.Body.Outs[i].ToBytesArray())
				leaves = append(leaves, leafHash)
			}
			outsStateMerkleTree := merkle.NewFullMerkleTree(leaves...)
			if outsStateMerkleTree.MerkleRoot.String() != b.Header.OutsMerkleRoot.String() {
				return fmt.Errorf("tree root is not correct")
			}
			err = tools.tree.HookSecondLayerTree(outsStateMerkleTree)
			if err != nil {
				return err
			}
			tools.ledgerInfo.SetShardHeight(b.Header.ShardIndex, b.Header.Height)
		}
	}

	tools.height++
	if err := tools.verifyLedger(&block.Header.ShardLedgerMerkleRoot); err != nil {
		return err
	}
	fmt.Printf("tools receive a block at height %d\n", block.Header.Height)
	fmt.Printf("tools tree root is %v\n", tools.tree.MerkleRoot)
	return nil
}

func (tools *testTools) verifyLedger(ledgerRoot *merkle.MerkleHash) error {
	tree := tools.tree
	treePath, err := tree.GetLastNodeMerklePath()
	treeSize := tree.Size
	if err != nil {
		fmt.Println("Failed to get last node path")
		return err
	}
	return merkle.VerifyLastNodeMerklePath(ledgerRoot, treeSize, treePath)
}

func (tools *testTools) handleMessage(message wire.Message) {
	switch msg := message.(type) {
	case *wire.MsgFetchInit:
		fmt.Println("tools receive fetch init message")
		ledger := tools.ledgerInfo
		shardHeght := tools.height
		tree := tools.tree

		treePath, err := tree.GetLastNodeMerklePath()
		treeSize := tree.Size
		if err != nil {
			fmt.Println("Failed to get last node path")
			return
		}

		var deposits []*wire.OutWithProof
		if tools.height <= 1 {
			path, err := tools.tree.GetLastNodeMerklePath()
			if err != nil {
				tools.t.Error(err)
			}
			deposits, err = getDepositsFromGenesisBlock(tools.shard, msg.Address, path)
			if err != nil {
				tools.t.Error(err)
			}
		}

		latestHeader := tools.blockchain.GetShardsHeaderByHeight(tools.shard, wire.BlockHeight(shardHeght))
		result := &wire.MsgReturnInit{
			ShardIndex:   tools.shard,
			MsgID:        msg.MsgID,
			Ledger:       ledger,
			RightPath:    *treePath,
			ShardHeight:  shardHeght,
			LatestHeader: *latestHeader,
			TreeSize:     treeSize,
			Deposits:     deposits,
		}
		fmt.Println("tools returns init message")
		tools.abci.onNewMessage(result)
	case *wire.MsgFetchTxs:
		fmt.Println("tools receive fetchtx message")
		var tx *wire.MsgTx
		var err error
		switch tools.txGenType {
		case "transfer":
			tx, err = tools.txGen.makeTransferTx(tools.shard, tools.config.to, tools.config.value)
		case "deposit":
			tx, err = tools.txGen.makeDepositTx(tools.shard, tools.config.to, tools.config.binding, tools.config.value)
		}
		if err != nil {
			tools.t.Error(err)
			return
		}

		txWithProof := wire.MsgTxWithProofs{
			Tx:     *tx,
			Proofs: []merkle.MerklePath{},
		}

		for _, txIn := range tx.TxIn {
			outState := txIn.PreviousOutPoint.ToUnspentOutState()
			merkleHash := merkle.ComputeMerkleHash(outState.ToBytesArray())
			merklePath, err := tools.tree.GetMerklePath(merkleHash)
			if err != nil {
				tools.t.Error(err)
				return
			}
			txWithProof.Proofs = append(txWithProof.Proofs, *merklePath)
		}

		var rtnTxs []*wire.MsgTxWithProofs
		rtnTxs = append(rtnTxs, &txWithProof)
		returnMsg := &wire.MsgReturnTxs{
			MsgID:              msg.MsgID,
			ShardIndex:         msg.ShardIndex,
			Txs:                rtnTxs,
			SmartContractInfos: []*wire.SmartContractInfo{},
		}
		fmt.Println("tools returns fetchtx message")
		tools.abci.onNewMessage(returnMsg)
	}
}

func makeDepositData(height int64, t *testing.T) []byte {
	d := &wire.MTVDepositData{
		Height:         height,
		Value:          new(big.Int).SetInt64(0),
		BindingAddress: multivacaddress.Address{},
	}
	data, err := rlp.EncodeToBytes(d)
	if err != nil {
		t.Error(err)
	}
	return data
}

func makeWithdrawData(height int64, t *testing.T) []byte {
	d := &wire.MTVWithdrawData{
		WithdrawHeight: height,
		Value:          new(big.Int).SetInt64(0),
	}
	data, err := rlp.EncodeToBytes(d)
	if err != nil {
		t.Error(err)
	}
	return data
}

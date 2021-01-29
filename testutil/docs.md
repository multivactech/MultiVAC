## 测试桩

在本文件夹下我们提供了方便单元测试使用的测试桩，目前实现的对象有：
- blockchain：mockblockchain.go
- heartbeat:mockheart.go
- abci：fakechain.go

### fakechain
`fakechain`模拟了一个不需要共识的区块链，他维护了区块关系、交易以及账本状态，方便了需要进行账本更新的单元测试的编写

#### 创建
创建fakechain，需要使用`*testing.T`和`iblockchain.BlockChain`对象作为参数，例如：
```go
chain := testutil.NewMockChain("deposit")
fc := testutil.NewFakeChains(t,chain)
```
在上面的代码中，chain使用了本包中的MockChain，具体细将在下文介绍

#### 获得区块

方便的是，fakechain可以轻松获得区块，并且自动建立区块之间的连接关系，有以下两种方式可以获得区块：

```go
// 获取一个包含默认交易的区块
block1 := fc.Next(shard0, address, value)
// 获取一个给定交易的区块
block2 := fc.GetNewBlock(shard0, txs)
```

#### 获得交易

如果我们需要对交易进行操作，那么也提供了交易获取的方法：

```go
tx := fc.GetNeWTx(shard0, txType, txParams, unUsedOuts...)
```

需要注意的是，参数`unUsedOuts`是可选的，这就意味着你可以自己构建一个有效的Out，也可以使用fakechain提供的默认Out，这些默认的Out都是硬编码在创始区块中的。

这种场景适用于：当下一个区块需要使用上一个区块中的交易，例如：

```go
block1 := fc.Next(shard0, address, value)

tx := fc.GetNewTx(shard0, txType, txParams, block1.body.Outs[0])
block2 := fc.GetNewBlock(shard0, tx)
```

#### 获得State

有时候我们需要获取账本的状态，可以通过`GetState`函数获取当前账本的state对象

```go
state := fc.GetState()
```

注意：每生成一个块，state就会变化

#### 获得Path

通过`GetOutPath`可以获得一个有效Out的默克尔路径

```go
path := fc.GetOutPath(out)
```

#### 获得默克尔树

除了state变量可以获得账本信息，fakechain还提供了完整的默克尔树，通过分片编号，我们可以很容易得到它：

```go
tree := fc.Chains[shard0].ShardLedgerTree
```

注：ShardLedgerTree是一颗“完整”的树，因此我们可以获得树上的任何信息

### mockblockchain

MockChain是一个实现了iblockchain.BlockChain接口的测试桩，用于方便测试用例的编写，我们在MockChain中规定了一个基本的规范，用于不同的测试模块对该模块的使用

#### Topic

MockChain维护了一个主题变量，用于识别它将服务于哪个测试模块。同时，在MockChain的函数实现中，用Topic来区分部分主题下的执行逻辑和返回结果：

```go
func (chain *MockChain) ReceiveBlock(block *wire.MsgBlock) bool {
	switch chain.topic {
	// testing for deposit moudle
	case Deposit:
		shard := block.Header.ShardIndex
		height := wire.BlockHeight(block.Header.Height)
		if _, ok := chain.chasche[shard]; !ok {
			chain.chasche[shard] = make(map[wire.BlockHeight]*wire.MsgBlock)
		}
		chain.chasche[shard][height] = block
		return true
	}
	return false
}
```

所以当我们为某模块编写测试用例时，如果需要MockChain作为测试桩，那么首先需要在`package.go`定义模块名称映射，然后在相应的函数中的switch case选择分支中编写对应逻辑，在对象的创建上上，我们也需要用主题来构建，例如：

```go
chain := testulit.NewMockChain("deposit")
```
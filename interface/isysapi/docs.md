## 交易类型说明

该文档用于说明不同类型交易的参数及介绍，持续更新

### 普通交易
符合`Utxo`模型的区块链交易，由输入（txIn）和输出（txOut）构成，txIn是前一笔交易的txOut，在下文中普通交易产生的txOut统称为`Out`

#### TransferParams
- To：交易发送的目标账户地址
- Shard：交易发送的目标分片号
- Amount：交易的发送金额

TransferParams是交易的基本参数，任何扩展交易类型必须包含上述三个字段

### 抵押交易
扩展自普通交易，输入为`Out`，生成特殊的抵押Out（以下简称`Dout`）用于用户抵押资金从而获得挖矿证明。

#### DepositParams
- BindingAddress：抵押绑定的用户地址

除了基本的交易参数外，BindingAddress用于记录该抵押交易所绑定的用户地址，该用户地址将会被授予挖矿权利

### 取款交易
扩展自普通交易，输入为`Dout`，生成特殊的取款Out（以下简称`Wout`）用于用户放弃抵押交易。

#### WithdrawParams
同普通交易参数，无任何特殊限定

### 绑定交易
扩展自普通交易，输入为`Dout`，生成一个新的`Dout`，用于用户修改之前生效的抵押交易中的绑定地址。

#### BindingParams
- BindingAddress：新的绑定地址
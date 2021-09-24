# 扇区修复

Filecoin在封装或挖矿过程中，可能面临扇区数据丢失，那么就要被销毁PreCommit预质押的FIL，或者终止扇区最大损失扇区的90天的收益。扇区修复能修复丢失的文件，来减少或者避免损失。

## 扇区丢失的原因

### 1.存储盘坏盘
矿商为了降低封装成本，不得不使用裸盘做存储，来降低成本，提高自己的竞争力，往往会直接使用裸盘做扇区的存储。
16T的盘，可以存储130多个32GiB扇区，如果损坏一个硬盘，数据无法恢复要终止扇区，最大损失扇区90天的全网平均收益。

### 2. NVMe缓存盘坏盘
   在这个情况下，扇区有2个状态会造成损失。
   - 扇区已经提交了PreCommit消息，但是30内未提交ProveCommit消息，会被销毁PreCommit预质押的FIL；
   - 设置 FinalizeEarly=false，使用先提交ProveCommit再落到存储，等同丢失扇区需要终止扇区。

## 扇区修复的工作原理

从原因可以分析，所有需要恢复的扇区都是已经提交过PreCommit消息之后的扇区，一旦数据丢失，只能重新组装原始封装的数据，重新封装扇区。

```go
p1o, err := ffi.SealPreCommitPhase1(
    sector.ProofType,
    paths.Cache,
    paths.Unsealed,
    paths.Sealed,
    sector.ID.Number,
    sector.ID.Miner,
    ticket,
    pieces,
)

p2...
```
重新封装CC扇区（无订单），pieces使用官方默认的生成方式即可，额外的需要获取ticket和ProofType。

这是链上记录的PreCommit消息体：
```json
{
  "RegisteredProof": 9,
  "SectorNumber": 1322006,
  "SealedCid": "bagboea4b5abcasqanjadumno7blvgx4k5pk765cki6vurnpgs2q3trt2trkznhj3",
  "SealRandEpoch": 925221,
  "DealIds": [],
  "Expiration": 2480426,
  "ReplaceCapacity": false,
  "ReplaceSectorDeadline": 0,
  "ReplaceSectorPartition": 0,
  "ReplaceSector": 0
}
```
ProofType在不通网络版本和扇区大小下是不相同的，[详细代码]( https://github.com/filecoin-project/lotus/blob/7a38cd9286fbe8c4faf7b1f4737b6ff4dd94d011/chain/actors/builtin/miner/miner.go#L263 )。直接使用消息体中的RegisteredProof更加方便。

通过扇区区ticket的高度SealRandEpoch，向链服务器再次提取出随机数。
```go
ticket, err := fullNodeApi.ChainGetRandomnessFromTickets(ctx, ts.Key(), crypto.DomainSeparationTag_SealRandomness, ticketEpoch, buf.Bytes())
if err != nil {
    return nil, nil, err
}
```

#### Go

构建filecoin-sealer-recover，你需要安装[Go 1.16.4 or higher](https://golang.org/dl/):

```bash
wget -c https://golang.org/dl/go1.16.4.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

### 构建
构建需要下载一些Go模块。这些通常托管在Github上，而Github来自中国的带宽较低。要解决此问题，请在运行之前通过设置以下变量来使用本地代理：
```bash
export GOPROXY=https://goproxy.cn,direct  
```

Build and install
```base
make clean all

sudo make install
```

### 使用方式
```base
export FULLNODE_API_INFO=链节点的token
sealer-recover --miner=f01000 --sectorNum=0 --sector-size=32GiB --sealing-result=/sector --sealing-temp=/temp
```
#### 参数介绍
| 参数 | 含义 | 备注 |
| :-----| :----- | :----- |
| miner | 需要修复扇区的矿工号  | 必填 |
| sectorNum | 需要修复的扇区号 | 必填 |
| sector-size | 需要修复的扇区大小 | 默认值: 32GiB |
| sealing-result | 修复后的扇区产物路径 | 默认值: ~/sector |
| sealing-temp | 修复过程的中间产物路径，需要大空间，建议使用NVMe盘 | 默认值: ~/temp <br/> 最小空间: <br/> 32GiB # > 512GiB! <br/> 64GiB  # > 1024GiB! |

### 优化
使用自己修改的lotus进行打包，能优化修复速度

### TODO
扇区恢复支持有订单的扇区，通过链的订单id，检索到订单，重新生成pieces。

## License

licensed under [Apache 2.0](https://github.com/froghub-io/filecoin-sealer-recover/blob/main/LICENSE)

# 扇区修复

Filecoin在封装或挖矿过程中，可能面临扇区数据丢失，那么就要被销毁PreCommit预质押的FIL，或者终止扇区最大损失扇区的90天的收益。扇区修复能修复丢失的文件，来减少或者避免损失。

## 扇区丢失的原因

### 1.存储盘坏盘
矿商为了降低封装成本，不得不使用裸盘做存储，来降低成本，提高自己的竞争力，往往会直接使用裸盘做扇区的存储。
16T的盘，可以存储130多个32GiB扇区，如果损坏一个硬盘，数据无法恢复要终止扇区，最大损失扇区90天的全网平均收益。

### 2. NVMe缓存盘坏盘
   在这个情况下，扇区有2个状态会造成损失。
   - 扇区已经提交了PreCommit消息，但是30内未提交ProveCommit消息，会被销毁PreCommit预质押的FIL；
   - 设置 `FinalizeEarly=false`，使用先提交ProveCommit再落到存储，等同丢失扇区需要终止扇区。

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

比较链上记录的PreCommit消息体中的SealedCid和修复程序PreCommit2计算结果storage.SectorCids，如果结果cid一致表示修复成功！


#### Go

构建filecoin-sealer-recover，你需要安装[Go 1.16.4 or higher](https://golang.org/dl/):

```shell
wget -c https://golang.org/dl/go1.16.4.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

### 构建
1、构建需要下载一些Go模块。这些通常托管在Github上，而Github来自中国的带宽较低。要解决此问题，请在运行之前通过设置以下变量来使用本地代理：
```shell
export GOPROXY=https://goproxy.cn,direct  
```

2、根据您的 CPU 型号，根据您的需要选择环境变量：

a.如果您有AMD Zen 或 Intel Ice Lake CPU（或更高版本），请通过添加以下两个环境变量来启用 SHA 扩展的使用：
```shell
export RUSTFLAGS="-C target-cpu=native -g"
export FFI_BUILD_FROM_SOURCE=1
```
有关此过程的更多详细信息，请参阅本地 Filecoin FFI 部分

b.一些没有 ADX 指令支持的老式 Intel 和 AMD 处理器可能会因为非法指令错误而紊乱。要解决这个问题，添加 CGO_CFLAGS 环境变量:
```shell
export CGO_CFLAGS_ALLOW="-D__BLST_PORTABLE__"
export CGO_CFLAGS="-D__BLST_PORTABLE__"
```
c.默认情况下，证明库中使用“multicore-sdr”选项。 除非明确禁用，否则此功能也用于 FFI。 要禁用“multicore-sdr”依赖项的构建，请将“FFI_USE_MULTICORE_SDR”设置为“0”：
```shell
export FFI_USE_MULTICORE_SDR=0
```

3、Build and install
```shell
# 扇区恢复不区分mainnet或者calibnet
make clean all

sudo make install
```
将 `sealer-recover` 安装到 `/usr/local/bin`

4、安装完成后，使用下面的命令确保为正确的网络成功安装了.
```shell
sealer-recover --version
```

### 使用方式
help
```base
sealer-recover -h
```

启动：
```base
export FIL_PROOFS_USE_MULTICORE_SDR=1
export FIL_PROOFS_MAXIMIZE_CACHING=1
export FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1
export FIL_PROOFS_USE_GPU_TREE_BUILDER=1

export FULLNODE_API_INFO=链节点的token
sealer-recover --miner=f01000 \
    --sectors=0 \ 
    --sectors=1 \ 
    --sectors=2 \     
    --parallel=6 \ 
    --sealing-result=/sector \ 
    --sealing-temp=/temp
```
#### 参数介绍
| 参数 | 含义 | 备注 |
| :-----| :----- | :----- |
| miner | 需要修复扇区的矿工号  | 必填 |
| sectors | 需要修复的扇区号 | 必填 |
| parallel | 修复扇区p1的并行数, _参考核心数进行设置_ | 默认值：1 |
| sealing-result | 修复后的扇区产物路径 | 默认值: ~/sector |
| sealing-temp | 修复过程的中间产物路径，需要大空间，建议使用NVMe盘 | 默认值: ~/temp <br/> 最小空间: <br/> 32GiB # > 512GiB! <br/> 64GiB  # > 1024GiB! |

### 优化
使用自己修改的lotus进行打包，能优化修复速度

### TODO
- 支持有订单的扇区，通过链的订单id，检索到订单，重新生成pieces。

### Contributing
欢迎PR、错误报告和问题建议！对于重大更改，请先在issues中提出问题，以便讨论兼容性和收益。

### Other
- [FILFrog](https://www.froghub.io/)
- [Lotus CloudC2](https://github.com/froghub-io/lotus-cloudc2)

## License

Licensed under [Apache 2.0](https://github.com/froghub-io/filecoin-sealer-recover/blob/main/LICENSE)

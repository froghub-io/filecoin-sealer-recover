中文请点击[这里](./README_CN.md)
# Sector Repair

Filecoin may face sector data loss during sealing or mining, and then destroy PreCommit pre-pledged FILs, or terminate sectors with a maximum loss of 90 days of sector reward. Sector repair can repair the lost files to reduce or avoid the loss.

## Causes of sector loss

### 1. Bad storage disk

Miners are forced to use bare disks for storage to reduce costs and improve their competitiveness in order to reduce packaging costs, and will often use bare disks directly for sector storage.
16T disk, can store more than 130 sectors of 32GiB, if the hard disk is damaged, data can not be recovered and have to terminate the sector, the maximum loss is 90-day of average revenue of sectors across the network.

### 2. Bad NVMe cache disk

In this case, there are 2 states of the sector that can cause a loss.

- A sector that has submitted a PreCommit message, but has not submitted a ProveCommit message within 30 days will have its PreCommit pre-pledge FIL destroyed.

- Setting `FinalizeEarly=false`, using ProveCommit first before dropping to storage, is equivalent to losing a sector requiring termination of the sector.

## How sector repair works

From the reasons can be analyzed, all sectors that need to be recovered are sectors that have already been submitted PreCommit message, once the data is lost, only the original sealing data can be reassembled and resealing the sectors .

```go
p1o, err := ffi.SealPreCommitPhase1(
    sector.ProofType,
    paths.Cache,
    paths.Unsealed,
    paths.Sealed,
    sector.ID.Number,
    sector.ID.Miner,
    ticket,
    ID.Number, sector.ID.Miner, ticket, pieces,
)

p2...
```

Resealing the CC sector (without deals), pieces is just generated using the official default method, Only the ticket and ProofType need to be obtained additionally

This is the PreCommit message body recorded on the chain.

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

ProofType is different for different network versions and sector sizes, [detailed code]( https://github.com/filecoin-project/lotus/blob/7a38cd9286fbe8c4faf7b1f4737b6ff4dd94d011/chain/actors/builtin/miner/miner.go#L263 ). It is more convenient to use the RegisteredProof directly in the message body.

Using the SealRandEpoch in the precommit message body as the sector's ticketEpoch, the random number is extracted again from the chain server.

```go
ticket, err := fullNodeApi.StateGetRandomnessFromTickets(ctx, crypto.DomainSeparationTag_SealRandomness, ticketEpoch, buf.Bytes(), ts.Key ())
if err ! = nil {
    return nil, nil, err
}
```

Compare the SealedCid in the PreCommit message body recorded on the chain with the result of the repair procedure PreCommit2 calculation storage.SectorCids, if the result cid is the same the repair is successful!

### Install Go

To build filecoin-sealer-recover, you need to install [Go 1.16.4 or higher](https://golang.org/dl/):

```shell
wget -c https://golang.org/dl/go1.16.4.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

### Build

1. To build you need to download some Go modules. These are usually hosted on Github, which has a low bandwidth from China. To work around this, use a local proxy before running by setting the following variables.

```shell
export GOPROXY=https://goproxy.cn
```

2. Depending on your CPU model, you will want to export additional environment variables:

a. If you have **an AMD Zen or Intel Ice Lake CPU (or later)**, enable the use of SHA extensions by adding these two environment variables:

```shell
export RUSTFLAGS="-C target-cpu=native -g"
export FFI_BUILD_FROM_SOURCE=1
```

See the [Filecoin FFI section](https://docs.filecoin.io/get-started/lotus/installation/#native-filecoin-ffi) for more details about this process.

b. Some older Intel and AMD processors without the ADX instruction support may panic with illegal instruction errors. To solve this, add the `CGO_CFLAGS` environment variable:

```shell
    export CGO_CFLAGS_ALLOW="-D__BLST_PORTABLE__"
    export CGO_CFLAGS="-D__BLST_PORTABLE__"
```

c. By default, a 'multicore-sdr' option is used in the proofs library.  This feature is also used in FFI unless explicitly disabled.  To disable building with the 'multicore-sdr' dependency, set `FFI_USE_MULTICORE_SDR` to `0`:

```shell
 export FFI_USE_MULTICORE_SDR=0
```

3. Build and install

```shell
# Sector recovery does not distinguish between mainnet or calibnet
make clean all

sudo make install
```

Install `sealer-recover` to `/usr/local/bin`

4. Once the installation is complete, use the following command to ensure that the installation is successful for the correct network.

```shell
sealer-recover --version
```

### Usage

help

```shell
sealer-recover -h
```

To start.

```shell
export FIL_PROOFS_USE_MULTICORE_SDR=1
export FIL_PROOFS_MAXIMIZE_CACHING=1
export FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1
export FIL_PROOFS_USE_GPU_TREE_BUILDER=1

export FULLNODE_API_INFO=token of chain node
sealer-recover --miner=f01000 \
    --sectors=0 \ 
    --sectors=1 \ 
    --sectors=2 \     
    --parallel=6 \ 
    --sealing-result=/sector \ 
    --sealing-temp=/temp
```

#### Parameter Description

| Parameters | Meaning | Remarks |
| :-----| :----- | :----- |
| miner | The miner number of the sectors to be repaired | Required |
| sectors | Sector number to be repaired | required
| parallel | Number of parallel sectors to be repaired for sector p1, _refer to core count for setting_ | default: 1 |
| sealing-result | path to the repaired sector product | default: ~/sector |
| sealing-temp | Intermediate product path for repair process, large space required, NVMe disk recommended | Default: ~/temp <br/> Minimum space: <br/> 32GiB # > 512GiB! <br/> 64GiB # > 1024GiB!

### Optimization

Use your own modified lotus for packing, which optimizes the repair speed

### TODO

- Support for sectors with deals, retrieving the deals by the deals id of the chain and regenerating the pieces.

### Contributing

PRs, bug reports and issue suggestions are welcome! For major changes, please ask questions in issues first so that compatibility and benefits can be discussed.

### Other

- [FILFrog](https://www.froghub.io/)
- [Lotus CloudC2](https://github.com/froghub-io/lotus-cloudc2)

## License

Licensed under [Apache 2.0](https://github.com/froghub-io/filecoin-sealer-recover/blob/main/LICENSE)

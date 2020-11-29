
[//]: <>  (TODO tianhongce: Complete this document)

# shardprocessor
## resharding
**Detemine the next shard at H and work in the new shard at H+k** <br/>
**Between the K rounds, it need to create the new shard network connection and init the ledger of new shard**
* At height H: determine the next shard 
    - before resharding in this shard, after reshading in this shard: 
      1. Do nothing
    - before resharding in this shard, after reshading not in this shard:
      1. Set consensus stopped at height h+k, Since the consensus begins directly after the next round, it is necessary to set the consensus in the resharding round and stop the next round if it needs to stop.
    - before resharding not in this shard, after reshading in this shard:
      1. Set processor enabled
      2. Init ledger
      3. Init ledger complete, set the miner processor status: preparation
      4. Cache the block proposed by others miners
      5. After receive the confirmation of the cache block, verify and accept the block
      6. Broadcast the new enabledshards
    - before resharding not in this shard, after reshading not in this shard:
      1. Do nothing
* At height H+K: work in the new shard
    - before resharding in this shard, after reshading in this shard: 
      1. Update the proof
    - before resharding in this shard, after reshading not in this shard:
      1. Consensus stopped at height h+k by itself
      2. Broadcast the new enabledshards
    - before resharding not in this shard, after reshading in this shard:
      1. Start the next round after receive the confirmation msg
    - before resharding not in this shard, after reshading not in this shard:
      1. Do nothing

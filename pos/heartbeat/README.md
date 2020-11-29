# Package of heartbeat

`heartbeat` package is used to manage the heartbeat message of miner by HeartBeatManager. HeartBeatManager is responsible for sending and receiving heartbeat messages, and it maintains a white list of legal miners, all of whom can participate in consensus vote round.

## Rules

There are some rules for heartbeat:

- The timestamp of each heartbeat cannot be the same
- The signature of each heartbeat can be verified
- The heartbeat must contains a proof of miner
- The interval between sending heartbeats can't be too long
- The timestamp in the heartbeat cannot differ too much from the local time.

And for a miner, **the online 30min(Time can be changed) condition must be met**

All time rules can be set by parameters, the parameters are in the params package(`params/params.go`):

- `ONLINE_TIME_S`: The duration of a miner must be online

- `TIME_DEVIATION_S`: The acceptable deviation of time for heartbeat

- `SEND_HEARTBEAT_INTERVAL_S`: The interval to send heartbeat

## White list

All the miners in the white list are considered legal. The miner who wants to enter the white list must abide by the following rules：

- Make sure online is more than 30min
- Send heartbeat frequently


For testnet

For testnet, the first batch of miners do not need to wait for a while, so we can adjust the time parameter by rpc.

The interface is `SetHeartBeatTime`, it need three parameters:

- `online` to set the miner should online time
- `deviation` to set the acceptable deviation of time for heartbeat
- `interval` to set interval to send heartbeat
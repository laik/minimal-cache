# example
```
cd bin
./minimal-cache-server -raft 127.0.0.1:10000 -tcp 127.0.0.1:10001 -http 127.0.0.1:10002 -datadir tmp/node1 
./minimal-cache-server -raft 127.0.0.1:20000 -tcp 127.0.0.1:20001 -http 127.0.0.1:20002 -datadir tmp/node2 -join 127.0.0.1:10001
./minimal-cache-server -raft 127.0.0.1:30000 -tcp 127.0.0.1:30001 -http 127.0.0.1:30002 -datadir tmp/node3 -join 127.0.0.1:10001
```
# minimal-cache

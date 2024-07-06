#/bin/sh

# Limit all incoming and outgoing network to 30mbit/s
tc qdisc add dev eth0 handle 1: ingress
tc filter add dev eth0 parent 1: protocol ip prio 50 u32 match ip src 0.0.0.0/0 police rate 30mbit burst 10k drop flowid :1
tc qdisc add dev eth0 root tbf rate 30mbit latency 25ms burst 10k`
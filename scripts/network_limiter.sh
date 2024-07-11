#!/bin/sh

# docker environment variable
node_id=$node_num
echo node_id is $node_id

case $node_id in 
    0)
    down=10.0mbit
    up=10.0mbit
    ;;
    1)
    down=20.0mbit
    up=20.0mbit
    ;;
    2)
    down=30.0mbit
    up=30.0mbit
    ;;
    3)
    down=40.0mbit
    up=40.0mbit
    ;;
    4)
    down=50.0mbit
    up=50.0mbit
    ;;
    5)
    down=60.0mbit
    up=60.0mbit
    ;;
    6)
    down=70.0mbit
    up=70.0mbit
    ;;
    7)
    down=80.0mbit
    up=80.0mbit
    ;;
    8)
    down=90.0mbit
    up=90.0mbit
    ;;
    9)
    down=100.0mbit
    up=100.0mbit
    ;;
    *)
    down=1.0mbit
    up=1.0mbit
    ;;
esac

# Limit all incoming and outgoing network to 30mbit/s
tc qdisc add dev eth0 handle 1: ingress
tc filter add dev eth0 parent ffff: protocol ip prio 50 u32 match ip src 0.0.0.0/0 police rate $down burst 10k drop flowid :1 # download
tc qdisc add dev eth0 root tbf rate $up latency 10ms burst 10k # upload
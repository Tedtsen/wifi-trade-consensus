#!/bin/sh

# docker environment variable
node_id=$node_num
is_faulty=$is_faulty
echo node_id is $node_id

#
#  tc uses the following units when passed as a parameter.
#  kbps: Kilobytes per second
#  mbps: Megabytes per second
#  kbit: Kilobits per second
#  mbit: Megabits per second
#  bps: Bytes per second
#       Amounts of data can be specified in:
#       kb or k: Kilobytes
#       mb or m: Megabytes
#       mbit: Megabits
#       kbit: Kilobits
#  To get the byte figure from bits, divide the number by 8 bit
#

case $node_id in 
    0)
    down=10mbps
    up=10mbps
    burst=2000K
    ;;
    1)
    down=20mbps
    up=20mbps
    burst=4000K
    ;;
    2)
    down=30mbps
    up=30mbps
    burst=6000K
    ;;
    3)
    down=40mbps
    up=40mbps
    burst=8000K
    ;;
    4)
    down=50mbps
    up=50mbps
    burst=10000K
    ;;
    5)
    down=60mbps
    up=60mbps
    burst=12000K
    ;;
    6)
    down=70mbps
    up=70mbps
    burst=14000K
    ;;
    7)
    down=80mbps
    up=80mbps
    burst=16000K
    ;;
    8)
    down=90mbps
    up=90mbps
    burst=18000K
    ;;
    9)
    down=100mbps
    up=100mbps
    burst=20000K
    ;;
    *)
    down=1mbps
    up=1mbps
    burst=100K
    ;;
esac

if [ "$is_faulty" = true ]; then
    down=1mbps
    up=1mbps
    burst=100K
fi

echo is faulty ? $is_faulty
echo up is $up
echo down is $down
echo burst is $burst

# Limit all incoming and outgoing network to megabytes/s
tc qdisc add dev eth0 handle ffff: ingress
tc filter add dev eth0 parent ffff: protocol ip prio 50 u32 match ip src 0.0.0.0/0 police rate $down burst $burst flowid :1 # download
tc qdisc add dev eth0 root tbf rate $up latency 10ms burst 10k # upload
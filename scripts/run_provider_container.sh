#/bin/sh

# docker run -p 8080:8080 -e node_num=0 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node0 -td container-name
# docker run -p 192.168.0.109:8080:8080 -e node_num=0 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node0 -td node-0703
# docker run -p 192.168.0.109:8081:8081 -e node_num=1 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node1 -td node-0703
# docker run -p 192.168.0.109:8082:8082 -e node_num=2 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node2 -td node-0703
# docker run -p 192.168.0.109:8083:8083 -e node_num=3 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node3 -td node-0703
# docker run -p 192.168.0.109:8084:8084 -e node_num=4 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node4 -td node-0703
# docker run -p 192.168.0.109:8085:8085 -e node_num=5 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node5 -td node-0703
# docker run -p 192.168.0.109:8086:8086 -e node_num=6 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node6 -td node-0703
# docker run -p 192.168.0.109:8087:8087 -e node_num=7 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node7 -td node-0703
# docker run -p 192.168.0.109:8088:8088 -e node_num=8 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node8 -td node-0703
# docker run -p 192.168.0.109:8089:8089 -e node_num=9 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node9 -td node-0703

docker run -e node_num=0 -p 8080:8080 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node0 -td node-0703
docker run -e node_num=1 -p 8081:8081 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node1 -td node-0703
docker run -e node_num=2 -p 8082:8082 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node2 -td node-0703
docker run -e node_num=3 -p 8083:8083 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node3 -td node-0703
docker run -e node_num=4 -p 8084:8084 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node4 -td node-0703
docker run -e node_num=5 -p 8085:8085 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node5 -td node-0703
docker run -e node_num=6 -p 8086:8086 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node6 -td node-0703
docker run -e node_num=7 -p 8087:8087 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node7 -td node-0703
docker run -e node_num=8 -p 8088:8088 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node8 -td node-0703
docker run -e node_num=9 -p 8089:8089 --add-host=host.docker.internal:host-gateway --cap-add=NET_ADMIN --name node9 -td node-0703
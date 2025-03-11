---
title: Distributed

---

# Distributed Auction-based Network Selection Simulation
#### Paper
[placeholder-link](https://www.google.com)
#### Prerequisites
1. Simulation Code
2. [Docker Desktop](https://www.docker.com/products/docker-desktop/)
3. [iperf3](https://iperf.fr/iperf-download.php)
4. Custom Linux kernel for WSL 2
    :::warning
    This is very important as some `TC` and `QDISC` commands will fail to run if certain modules in the Linux kernel are not enabled.
    :::

This setup is only tested on Windows 11 running custom Linux kernel for WSL 2.

--- 
### Recompiling WSL 2 Linux Kernel
The compilation of kernel must be performed in a Linux environment (not on Windows), a vanilla WSL 2 environment can be used for this purpose.
1. Install tools necessary to compile the kernel:
`$ sudo apt update && sudo apt install build-essential flex bison libssl-dev libelf-dev bc python3 pahole
`
2. Download WSL 2 source code from github:
`$ git clone https://github.com/microsoft/WSL2-Linux-Kernel.git --depth=1 -b linux-msft-wsl-6.1.y`
3. Open source code directory:
`$ cd WSL2-Linux-Kernel`
4. Backup default `.config` file by copying it to `.config.old`:
`$ cp .config .config.old`
5. Edit config with text editor to enable the modules required:
`$ vim .config` 
In the text editor, append the text below to the existing lines.
    ```bashrc=
    # TC QDISC
    CONFIG_NET_SCHED=y
    CONFIG_NET_SCH_CBQ=y
    CONFIG_NET_SCH_HTB=y
    CONFIG_NET_SCH_CSZ=y
    CONFIG_NET_SCH_PRIO=y
    CONFIG_NET_SCH_RED=y
    CONFIG_NET_SCH_SFQ=y
    CONFIG_NET_SCH_TEQL=y
    CONFIG_NET_SCH_TBF=y
    CONFIG_NET_SCH_GRED=y
    CONFIG_NET_SCH_DSMARK=y
    CONFIG_NET_SCH_INGRESS=y
    CONFIG_NET_QOS=y
    CONFIG_NET_ESTIMATOR=y
    CONFIG_NET_CLS=y
    CONFIG_NET_CLS_TCINDEX=y
    CONFIG_NET_CLS_ROUTE4=y
    CONFIG_NET_CLS_ROUTE=y
    CONFIG_NET_CLS_FW=y
    CONFIG_NET_CLS_U32=y
    CONFIG_NET_CLS_RSVP=y
    CONFIG_NET_CLS_RSVP6=y
    CONFIG_NET_CLS_POLICE=y

    CONFIG_NET_SCH_HFSC=y
    CONFIG_NET_SCH_ATM=y
    CONFIG_NET_SCH_MULTIQ=y
    CONFIG_NET_SCH_NETEM=y
    CONFIG_NET_ACT_TUNNEL_KEY=y
    CONFIG_NET_ACT_POLICE=y
    CONFIG_NET_ACT_GACT=y
    CONFIG_DUMMY=y
    CONFIG_VXLAN=y
    ```
    **OPTIONAL**:
    - Edit `CONFIG_LOCALVERSION` to a custom image name so that it could be identified while running.
7. Build the kernel:
`$ make -j<insert-processor-count>`
    :::info
    Processor count dictates the number of computer cores used in the compilation process. Insert a higher number for a shorter compilation time.
    :::
7. Install the kernel modules and headers:
`$ sudo make modules_install headers_install`
8. *If the compilation process is done in WSL 2, copy the kernel image to the Windows file system:
`$ cp arch/x86/boot/bzImage /mnt/c/`
    :::info
    `/mnt/c/` represents the `C:/` drive on Windows.
    :::

### Installing Custom WSL 2 Linux Kernel

On Windows, WSL 2 can be configured to run a custom kernel by creating a custom configuration file.
1. Create or edit `.wslconfig`:
`$ notepad %USERPROFILE%\.wslconfig`
    In the text editor, insert the text below:
    ```bashrc=
    [wsl2]
    kernel=C:\\bzImage
    ```
    :::info
    `C:\\bzImage` is the filepath of the image file.
    :::
2. Shutdown WSL instance by typing the command below in a PowerShell terminal windows:
    `$ wsl --shutdown`

### Running the Kernel
The new kernel will run by default when launching the WSL 2 terminal.
1. Check the kernel version:
`$ uname -r`
The output should be the same as specified in  `CONFIG_LOCALVERSION`.
    
---

### Running Simulation (Provider)
#### 1. Run the command below in the directory where the Dockerfile is located to build a Docker image:
`$ docker build --no-cache . -t <image-name>`
:::info
This command instructs docker to build an image with the name tag `<image-name>`.
:::

The `Dockerfile` should look like the one below:
```bashrc=
# syntax=docker/dockerfile:1

FROM golang:1.22

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o docker-build ./cmd/provider

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/reference/dockerfile/#expose
EXPOSE 8080-8089 10000-11000

RUN apt -y update
RUN apt -y install iperf3
RUN apt -y install iproute2


# RUN bash ./app/scripts/network_limiter.sh

# Run
CMD ["sh", "-c", "/app/scripts/network_limiter.sh && /app/docker-build"]
# CMD [ "/app/docker-build" ]
```

#### 2. Set a new environment variable `IMAGE_NAME` in console:

`$ env:IMAGE_NAME="<image-name>"` 

:::info
This command sets a new custom console environment variable key:`IMAGE_NAME`, value:`<image-name>`.
:::

#### 3. Launch Docker containers according to `docker-compose` configuration file:

`$ docker compose up`

:::info
This command instructs docker to run the image according to the configurations in `docker-compose.yml`. The image name will be interpolated from the console environment variable.
:::

The `docker-compose.yml` should look like the one below:
```bashrc=
version: "3.7"
services:
  node0:
    container_name: "node0"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10010-10099:10010-10099"
      - "7777:7777"
      - "8080:8080"
    environment:
      node_num: "0"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node1:
    container_name: "node1"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10100-10199:10100-10199"
      - "8081:8081"
    environment:
      node_num: "1"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node2:
    container_name: "node2"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10200-10299:10200-10299"
      - "8082:8082"
    environment:
      node_num: "2"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node3:
    container_name: "node3"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10300-10399:10300-10399"
      - "8083:8083"
    environment:
      node_num: "3"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node4:
    container_name: "node4"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10400-10499:10400-10499"
      - "8084:8084"
    environment:
      node_num: "4"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node5:
    container_name: "node5"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10500-10599:10500-10599"
      - "8085:8085"
    environment:
      node_num: "5"
      is_faulty: "true"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node6:
    container_name: "node6"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10600-10699:10600-10699"
      - "8086:8086"
    environment:
      node_num: "6"
      is_faulty: "true"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node7:
    container_name: "node7"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10700-10799:10700-10799"
      - "8087:8087"
    environment:
      node_num: "7"
      is_faulty: "true"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node8:
    container_name: "node8"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10800-10899:10800-10899"
      - "8088:8088"
    environment:
      node_num: "8"
      is_faulty: "true"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
  node9:
    container_name: "node9"
    privileged: true
    image: ${IMAGE_NAME}
    ports:
      - "10900-10999:10900-10999"
      - "8089:8089"
    environment:
      node_num: "9"
      is_faulty: "true"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    cap_add:
      - ALL
```

#### 4. Check Docker Desktop container dashboard:
10 docker containers should be running, namely `node0`, `node1`, ..., `node9`.

---

### Troubleshooting Docker Network

#### 1. Unable to connect to container

Check if `host.docker.internal` resolves to the private ip of localhost:
`$ tracert host.docker.internal`

If the IP address is different or outdated, try the methods below:

- Rebuild docker image without cache:
`$ docker build --no-cache . -t <image-name>`
- Enable `Add the *docker.internal names...` in Docker General Settings:
![image](https://hackmd.io/_uploads/SJbPD40Jye.png)
- If the previous step doesn't work, manually check `/etc/hosts` file on Windows:
`$ notepad C:\Windows\System32\drivers\etc\hosts`
    - These lines should be appended to the file added by Docker:
        ```bashrc=
        # Added by Docker Desktop
        10.81.52.123 host.docker.internal
        10.81.52.123 gateway.docker.internal
        # To allow the same kube context to work on the host and the container:
        127.0.0.1 kubernetes.docker.internal
        # End of section
        ```
    - Check if the IP address assigned to `host.docker.internal` matches the host's local IP.
    - The IP should match after enabling the Docker Setting in the previous step. Otherwise, manually edit the IP entry in `/etc/hosts` as the last resort.

#### 2. Unstable connection caused by MTU mismatch 
The MTUs on Windows 11, WSL 2 and container should match, they typically have a value of 1500.
- Check MTU on Windows by typing the command below in the Windows terminal:
`$ netsh interface ipv4 show interfaces`
    - Check the MTU of vEthernet (WSL (Hyper-V firewall))
- Check MTU on WSL 2 by typing the command below in the distro:
`$ ip link show eth0`
- Check MTU in Docker container by typing the command below in the container's terminal:
`$ ip link show eth0`
- Disable RSC on Windows by typing the command below in a PowerShell terminal:
`Get-NetAdapterRSC | Disable-NetAdapterRsc`

Example output of `$ ip link show` where `eth0` has a `mtu` value of 1280:
```
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: dummy0: <BROADCAST,NOARP> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether 6a:70:2d:19:5a:c5 brd ff:ff:ff:ff:ff:ff
3: teql0: <NOARP> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 100
    link/void
4: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1280 qdisc mq state UP mode DEFAULT group default qlen 1000
    link/ether 00:15:5d:15:a6:dc brd ff:ff:ff:ff:ff:ff
```

#### 3. Slow internet speed in Docker container
The issue might be caused by slow DNS lookup performed in the container, specifying DNS servers like Google DNS in the `/etc/resolv.conf` file might improve the performance.
- Create a file named `resolv.conf` in the same directory as the `docker-compose` file with the contents below:
```bashrc=
nameserver <local-ip>
nameserver 8.8.8.8
nameserver 8.8.4.4
```
:::info
8.8.8.8 and 8.8.4.4 are both Google DNS servers.
:::
- Add the following lines to `docker-compose` file:
```bashrc=
version: '2'
services:
  node0:
    container_name: "my_container"
    image: "my_image_name"
    networks:
      - "my_net"
    volumes:
      - "./resolv.conf:/etc/resolv.conf"
```
<!-- #### WSL config
[link](https://github.com/microsoft/WSL/issues/4901)
```
[wsl2]
memory=6GB
kernel=C:\\Working6

[experimental]
networkingMode=mirrored
``` -->

#### 4. Slow WSL 2 performance
Add WSL 2 distro to Windows Defender's exclusion list:
https://medium.com/@leandrocrs/speeding-up-wsl-i-o-up-than-5x-fast-saving-a-lot-of-battery-life-cpu-usage-c3537dd03c74

#### 5. More on Docker networking configurations
https://loadforge.com/guides/advanced-network-configuration-for-high-performance-docker-containers

---

### Running Simulation (Consumer)

#### Configurations
The configuration file is located in `./cmd/consumer/config.json`. It should look like:
```jsonld=
{
    "id": "consumer-id-1",
    "address": "10.81.52.123:9000",
    "iperf3_base_server_port": "5001",
    "iperf3_server_count": "0",
    "price": "0.5",
    "uplink": "30",
    "downlink": "50",
    "mu": "0.8",
    "delta": "1",
    "epsilon": "2",
    "output_dir": "C:/Dev/AUC/results",
    "tau": 1
}
```
The only fields that matter are **`address`** and **`output_dir`**, the rests are deprecated since the paramaters for `BUY` event are now controlled by the `trigger` process.
| Field Name | Type   | Description                                                           |
| ---------- | ------ | --------------------------------------------------------------------- |
| address    | string | IP address of the consumer process, must be the local IP of the host. |
| output_dir           | string       |  Directory to save the simulation results. The results will only be written to file when the process receives the interupt signal i.e. `Ctrl^c`. The filename is in the following format: `consumer_transactions--yyyy-MM-dd--HH-mm-ss`.                                                                   |



#### Running consumer GO process
`$ go run ./cmd/consumer`

---

### Running Simulation (Trigger)
The `trigger` process triggers the consumer events to be sent to the providers.
#### Configurations
The configuration file is located in `./cmd/trigger/config.json`. It should look like:
```jsonld=
{
    "strategy": "auction",
    "consumer_address": "10.81.52.123:9000",
    "buy_event_count": 100,
    "buy_event_interval_mean": 60,
    "buy_event_interval_std_dev": 10,
    "buy_event_interval_lowest": 30,
    "buy_event_interval_highest": 120,
    "uplink_mean": 25,
    "uplink_std_dev": 12.5,
    "uplink_lowest": 5,
    "uplink_highest": 50,
    "downlink_mean": 25,
    "downlink_std_dev": 12.5,
    "downlink_lowest": 5,
    "downlink_highest": 50,
    "price_mean": 0.5,
    "price_std_dev": 0.25,
    "price_lowest": 0.1,
    "price_highest": 1,
    "mu_mean": 0.8,
    "mu_std_dev": 0.2,
    "mu_lowest": 0.1,
    "mu_highest": 0.99,
    "delta_mean": 0.8,
    "delta_std_dev": 0.2,
    "delta_lowest": 0.1,
    "delta_highest": 0.99,
    "epsilon_mean": 2,
    "epsilon_std_dev": 1,
    "epsilon_lowest": 1,
    "epsilon_highest": 3,
    "flow_size_mean": 500,
    "flow_size_std_dev": 250,
    "flow_size_lowest": 100,
    "flow_size_highest": 1024,
    "provider_list": [
        {
            "provider_id": "mock-id-0",
            "address": "host.docker.internal:8080"
        },
        {
            "provider_id": "mock-id-1",
            "address": "host.docker.internal:8081"
        },
        {
            "provider_id": "mock-id-2",
            "address": "host.docker.internal:8082"
        },
        {
            "provider_id": "mock-id-3",
            "address": "host.docker.internal:8083"
        },
        {
            "provider_id": "mock-id-4",
            "address": "host.docker.internal:8084"
        },
        {
            "provider_id": "mock-id-5",
            "address": "host.docker.internal:8085"
        },
        {
            "provider_id": "mock-id-6",
            "address": "host.docker.internal:8086"
        },
        {
            "provider_id": "mock-id-7",
            "address": "host.docker.internal:8087"
        },
        {
            "provider_id": "mock-id-8",
            "address": "host.docker.internal:8088"
        },
        {
            "provider_id": "mock-id-9",
            "address": "host.docker.internal:8089"
        }
    ]
}
```


| Field Name | Type   | Description                  |
| ---------- | ------ | ---------------------------- |
| strategy   | string | Either `auction` or `naive`. |
| consumer_address           |    string    | The listening address of the `consumer` process. The `trigger` process will send `TRIGGER_BUY` event to the `consumer` process.                            |

Numerical fields accept normal distribution parameters as inputs to simulate randomized values for each `BUY` event. The parameters include: `mean`, `standard deviation`, `lowest value`, `highest value`.

`provider_list` is a list of provider info, the provider `address` has to be reachable from the `consumer` process. `provider_id` has no importance and can be arbitrarily named.

#### Running consumer GO process
`$ go run ./cmd/trigger`
:::info
The trigger process **must** be run after the consumer process.
:::stributed Auction-based Network Selection 
## Paper
## Experiment Setup
### Prerequisites
Docker, iperf3, github code
### Step-by-step setup

# Omil

Just a random name.

Small pieces of tools to monitor network status using simple ICMP sniffer and send metric data points to Time Series Database.

## Build

```shell script
git clone https://github.com/shallowclouds/omil.git && cd omil
chmod +x build.sh
./build.sh

# Simple script to enable Systemd Unit
sudo ./build/install.sh
```

## Usage

Fill the `conf/config.yml` configuration file or use `--config` to specify your configuration file.

```yaml
Hostname: "home-lab"
Targets:
  - Host: "baidu.com"
    Name: "baidu"
  - Host: "google.com"
    Name: "google"
  - Host: "10.1.1.1"
    Name: "gateway"
InfluxDBv2:
  Addr: "http://192.168.1.2:8086"
  Org: "org-name"
  Bucket: "bucket-name"
  Token: "token"
```

Run the omil binary:

> Sending and receiving ICMP packets needs root privilege.

```shell script
sudo /path/to/omil --config /etc/omil/config.yml
```

OR use systemd:
```shell script
sudo systemctl status omil.service
sudo systemctl start omil.service
sudo systemctl status omil.service
```

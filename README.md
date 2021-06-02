<p align=center><a href=""><img src=cmdo.svg?sanitize=true/></a></p>

---

Commando is a tiny tool that enables users to collect command outputs from a range of networking devices defined in an inventory file.

[![asciicast](https://asciinema.org/a/417792.svg)](https://asciinema.org/a/417792)


## Install
Using the sudo-less installation script makes it easy to download pre-built binary to the current working directory under `cmdo` name:

```bash
bash -c "$(curl -sL https://raw.githubusercontent.com/hellt/cmdo/master/get.sh)"
```

## Quickstart
1. Create an `inventory.yml` file with the devices information. An example [inventory.yml](inventory.yml) file lists three different platforms.
2. Run `./cmdo -i <path to inventory>`; the tool will read the inventory file and output the results of the commands in the `./
output` directory.

## Inventory file
The inventory file schema is simple, the network devices are defined under `.devices` element with each device identified by `<device-name>`:

```yaml
devices:
  <device1-name>:
  <device2-name>:
  <deviceN-name>:
```

Each device holds a number of options that define the device platform, auth parameters, and the commands to send:

```yaml
devices:
  <device1-name>:
   # platform is one of arista_eos, cisco_iosxe, cisco_nxos, cisco_iosxr,
   # juniper_junos, nokia_sros, nokia_sros_classic, nokia_srlinux
   platform: string 
      address: string
      username: string
      password: string
      send-commands:
         - cmd1
         - cmd2
         - cmdN
```

`send-commands` list holds a list of commands which will be send towards a device.

## Configuration options

* `-i <path>` - sets the path to the inventory file
* `-t` - appends the timestamp to the outputs directory, which results in the output directory to be named like `outputs_2021-06-02T15:08:00+02:00`.
* `-o file|stdout` - sets the output destination. Defaults to `file` which writes the results of the commands to the per-command files. If set to `stdout`, will print the commands to the terminal.

## Supported platforms
Commando leverages [scrapligo](https://github.com/scrapli/scrapligo) project to support the major network vendors:
* Arista EOS
* Cisco XR/XE/NXOS
* Juniper JunOS
* Nokia SR OS (MD-CLI and Classic)

In addition to that list, commando has the ability to add community provided scrapli drivers, such as:
* Nokia SR Linux - repo: [srlinux-scrapli](https://github.com/srl-labs/srlinux-scrapli)
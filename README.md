<p align=center><img src=cmdo_2.svg?sanitize=true/></p>

[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=424f35&labelColor=bec8d2)](https://goreportcard.com/report/github.com/hellt/cmdo)
[![Github all releases](https://img.shields.io/github/downloads/hellt/cmdo/total.svg?style=flat-square&color=424f35&labelColor=bec8d2)](https://github.com/hellt/cmdo/releases/)
---

Commando is a tiny tool that enables users to collect command outputs from a single or a multiple networking devices defined in an inventory file.

[![asciicast](https://asciinema.org/a/417792.svg)](https://asciinema.org/a/417792)


## Install
Using the sudo-less installation script makes it easy to download pre-built binary to the current working directory under `cmdo` name:

```bash
# for linux and darwin OSes
bash -c "$(curl -sL https://raw.githubusercontent.com/hellt/cmdo/master/get.sh)"
```

Windows users are encouraged to use WSL, but if it is not possible, the `.exe` file can be found in [Releases](https://github.com/hellt/cmdo/releases) section.

Linux users can leverage pre-built deb/rpm packages that are also available in the Releases section. Either download the package manually, or set `--use-pkg` flag with the install script:

```
bash -c "$(curl -sL https://raw.githubusercontent.com/hellt/cmdo/master/get.sh)" -- --use-pkg
```

## Quickstart
**If you want to run commands against multiple devices at once:**

1. Create an inventory file with the devices information. An example [inventory.yml](inventory.yml) file lists three different platforms.
2. Run `./cmdo -i <path to inventory>`; the tool will read the inventory file and output the results of the commands in the `./
output` directory.

**If you want to run commands against a single device:**

1. Use the CLI flags to define the device and commands to send:  
    `-a <address>` - IP/DNS name of the device  
    `-k <platform>` - one of the [supported](#supported-platforms) platforms  
    `-u <username> -p <password>` - SSH credentials  
    `-c <command1 :: command2 :: commandN>` - a single or a list of `::`-delimited commands  
  an example command could be:  
  `cmdo -o stdout -a clab-scrapli-srlinux -u admin -p admin -k nokia_srlinux -c "show version :: show system aaa"`


### Running commands in bulk
As indicated in the quickstart, `commando` can run commands against many devices as opposed to the _singe-device_ operation.

For the _bulk_ mode the devices are expressed in the sort-of inventory. The inventory file schema is simple, the network devices are defined under `.devices` element with each device identified by `<device-name>`:

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

`send-commands` list holds a list of commands which will be send towards a device. Check out the attached [example inventory](inventory.yml) file to a reference.

## Configuration options

* `--inventory | -i <path>` - sets the path to the inventory file
* `--add-timestamp | -t` - appends the timestamp to the outputs directory, which results in the output directory to be named like `outputs_2021-06-02T15:08:00+02:00`.
* `--output | -o value` - sets the output destination. Defaults to `file` which writes the results of the commands to the per-command files. If set to `stdout`, will print the commands to the terminal.
* `--filter | -f 'pattern'` - a filter to apply to device name to select the devices to which the commands will be sent. Can be a Go regular expression.

For the single-device operation mode the following flags must be used to define a device:
* `--address | -a <ip/dns>` - address of the device
* `--platform | -k <platform>` - one of the [supported](#supported-platforms) platform names
* `--username | -u <string>` - username
* `--password | -p <string>` - password
* `--command | -c <command1 :: commandN>` - list of commands to send, can be delimited with `::` to provide a list of commands

## Supported platforms
Commando leverages [scrapligo](https://github.com/scrapli/scrapligo) project to support the major network platforms:
| Network OS                       | Platform name                              |
| -------------------------------- | ------------------------------------------ |
| Arista EOS                       | `arista_eos`                               |
| Cisco XR/XE/NXOS                 | `cisco_iosxr`, `cisco_iosxe`, `cisco_nxos` |
| Juniper JunOS                    | `juniper_junos`                            |
| Nokia SR OS (MD-CLI and Classic) | `nokia_sros`, `nokia_sros_classic`         |

In addition to that list, commando has the ability to add community provided scrapli drivers, such as:
| Network OS     | Platform name                                                  |
| -------------- | -------------------------------------------------------------- |
| Nokia SR Linux | [`nokia_srlinux`](https://github.com/srl-labs/srlinux-scrapli) |

## Attributions
* Bullet icon is made by <a href="https://smashicons.com/" title="Smashicons">Smashicons</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a></div>
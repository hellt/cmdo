<p align=center><img src=cmdo_2.svg?sanitize=true/></p>

[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=424f35&labelColor=bec8d2)](https://goreportcard.com/report/github.com/glspi/cmdo)
[![Github all releases](https://img.shields.io/github/downloads/glspi/cmdo/total.svg?style=flat-square&color=424f35&labelColor=bec8d2)](https://github.com/glspi/cmdo/releases/)
---

** FORKED FROM: https://github.com/hellt/cmdo

Commando is a tiny tool that enables users 
* to collect command outputs from a single or a multiple networking devices defined in an inventory file
* send file-based or string-based configs towards the devices defined in the inventory file

all that with zero dependencies and a 1-click installation.

[![asciicast](https://asciinema.org/a/417792.svg)](https://asciinema.org/a/417792)


## Install
Using the sudo-less installation script makes it easy to download pre-built binary to the current working directory under `cmdo` name:

```bash
# for linux and darwin OSes
bash -c "$(curl -sL https://raw.githubusercontent.com/glspi/cmdo/master/get.sh)"
```

Windows users are encouraged to use WSL, but if it is not possible, the `.exe` file can be found in [Releases](https://github.com/glspi/cmdo/releases) section.

Linux users can leverage pre-built deb/rpm packages that are also available in the Releases section. Either download the package manually, or set `--use-pkg` flag with the install script:

```
bash -c "$(curl -sL https://raw.githubusercontent.com/glspi/cmdo/master/get.sh)" -- --use-pkg
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


## Inventory
As indicated in the quickstart, `commando` can run commands against many devices as opposed to the _singe-device_ operation.

For the _bulk_ mode the devices are expressed in the inventory file. The inventory file schema is simple, it consists of the following top-level elements:

```yaml
credentials: # container for credentials
transports:  # optional container for transport options
devices:     # here the devices connection details are
```

### Credentials
Commando let's you define many credential parameters which you can later associate with any of the devices. For example, a credential config for access switches might differ from the core routers.

```yaml
credentials:
  # this is a named credential config that you can refer to in the devices settings
  switches:
    username: admin
    password: admin
  routers:
    username: ops
    password: secret123

devices:
  sw1:
    address: some.host.com
    # credentials info from credentials containers named 'switches' will be used
    credentials: switches
  rtr1:
    address: some.host2.com
    # credentials info from credentials containers named 'routers' will be used
    credentials: routers
```

If you create a credential named `default`, then you can omit specifying credentials in the device configuration, this will be applied by default:

```yaml
credentials:
  default:
    username: admin
    password: admin

devices:
  sw1: # sw1 will use `default` credentials configuration
    address: some.host.com
```

Here is a full list of credentials configuration options:

```yaml
credentials:
  <name>:
    username:
    password:
    secondary-password:
    private-key: # takes a path to the private key
```

### Transports
Different transports can be defined in the inventory and mapped to the devices to support flexible connectivity options.

Transports are defined in the top level of the inventory:

```yaml
transports:
  myssh:
    port: 5622

devices:
  sw1: # sw1 will use port 5622 for SSH connection
    address: some.host.com
    transport: myssh
```

If the transport is not defined for a given device, the default transport options are assumed:

* port 22
* no strict host key checking
* transport type - standard
* ssh config file is not used

Here is a full list of transport configuration options:

```yaml
transports:
  <name>:
    port: # ssh port number to use
    strict-key: # true or false; sets host key checking
    transport-type: # `standard` or system. standard transport uses Go SSH client, `system` transport uses system's default SSH client (i.e. OpenSSH)
    ssh-config-file: # takes a path to ssh config file. Can only be used if transport is set to `system`
```

### Devices
The network devices are defined under `.devices` element with each device identified by a `<device-name>`:

```yaml
devices:
  <device1-name>:
  <device2-name>:
  <deviceN-name>:
```

Each device holds a number of options that define the device platform, its address, and the commands to send:

```yaml
devices:
  <device1-name>:
    # platform is one of arista_eos, cisco_iosxe, cisco_nxos, cisco_iosxr,
    # juniper_junos, nokia_sros, nokia_sros_classic, nokia_srlinux
    platform: string 
    address: string
    credentials: string # optional reference to the defined credentials
    transport: string # optional reference to the defined transport options
    send-commands-from-file: /path/to/file/with/show-commands.txt
    send-commands:
      - cmd1
      - cmd2
      - cmdN
    send-configs-from-file: /path/to/file/with/config-commands.txt
    send-configs:
      - cmd1
      - cmdN
    cfg-operations:
      # Note: cfg operations currently supported only on: arista_eos, cisco_iosxe,
      # cisco_nxos, cisco_iosxr, juniper_junos
      - type: load-config
        replace: false
        diff: true
        commit: false
        # Note: there is also a "config-from-file" option to load configurations from a file
        config: "interface loopback1\ndescription tacocat"
      - type: get-config
        source: running
```

`send-commands` list holds a list of non-configuration commands which will be send towards a device. A non configuration command is a command that doesn't require to have a configuration mode enabled on a device. A typical example is a `show <something>` command.  
Outputs from each command of a `send-commands` list will be saved/printed.

If you want to keep the commands in a separate file, then you can use `send-commands-from-file` element which takes a path to a said file. You can combine `send-commands` and `send-commands-from-file` in a single device.

In contrast with `send-commands*` options, it is possible to tell `commando` to send configuration commands. For that we have the following configuration elements:

* `send-configs` - takes a list of configuration commands and executes then without printing/saving any output the commands may return
* `send-configs-from-file` - does the same, but the commands are kept in a file.

Entering in the config mode is handled by commando, so your config commands doesn't need to have any `conf t` or `configure private` commands. Just remember to add the `commit` command if your device needs it.

The order these options are processed in:

1. "cfg" operations
2. send-configs-from-file
3. send-configs
4. send-commands-from-file
5. send-commands


Check out the attached [example inventory](inventory.yml) file for reference.

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
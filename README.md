# cmdo - commando
cmdo is a tiny program that demonstrates how [scrapligo](https://github.com/scrapli/scrapligo) module can be used to retrieve information from network devices using a simple inventory file.

## Install
build from source with `go build`

## Usage
1. Create an `inventory.yml` file, an example can be found in the repo.
2. Run `./cmdo`; the tool will read the `inventory.yml` file and output the results of commands in the terminal

example output:

```
$ ./cmdo

**************************
clab-scrapli-sros
**************************

-- show version:

TiMOS-B-20.10.R3 both/x86_64 Nokia 7750 SR Copyright (c) 2000-2021 Nokia.
All rights reserved. All use subject to applicable license agreements.
Built on Wed Jan 27 13:21:10 PST 2021 by builder in /builds/c/2010B/R3/panos/main/sros

-- show router interface:


===============================================================================
Interface Table (Router: Base)
===============================================================================
Interface-Name                   Adm       Opr(v4/v6)  Mode    Port/SapId
   IP-Address                                                  PfxState
-------------------------------------------------------------------------------
system                           Up        Down/Down   Network system
   -                                                           -
-------------------------------------------------------------------------------
Interfaces : 1
===============================================================================

**************************
clab-scrapli-ceos
**************************

-- show version:

 cEOSLab
Hardware version: 
Serial number: 
Hardware MAC address: 001c.7305.c8dd
System MAC address: 001c.7305.c8dd

Software image version: 4.25.0F-19436514.4250F (engineering build)
Architecture: x86_64
Internal build version: 4.25.0F-19436514.4250F
Internal build ID: 9271a36c-cfb6-4c58-971e-7e30a4eaf173

cEOS tools version: 1.1
Kernel version: 5.4.60-uksm

Uptime: 0 weeks, 0 days, 1 hours and 24 minutes
Total memory: 24630136 kB
Free memory: 19564168 kB

-- show uptime:

 10:20:31 up  1:31,  2 users,  load average: 0.83, 0.53, 0.45
```
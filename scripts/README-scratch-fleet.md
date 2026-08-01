# setup-scratch-fleet

`scripts/setup-scratch-fleet` sets up an isolated fleet-manager + hypervisor
stack on the dev VM so scratch-built images can be booted and reached end to
end, without touching the machine's real docker/imaginator networking.

## Purpose

The script mints its own TLS identities off the scratch-imaginator CA, brings
up a dedicated host-only bridge (`br@scratch`), starts a `hypervisor` and a
`fleet-manager` bound to that bridge, seeds a small IP pool, and gives you
`create-vm`/`destroy-vm` commands to boot images pulled from the scratch
imageserver. All state lives under `$SCRATCH_DIR` (default `~/scratch-fleet`)
and `teardown` removes it along with the bridge, leaving the dev VM's other
networking untouched.

## Prerequisites

- A running scratch-imaginator/imageserver (`scripts/setup-scratch-imaginator`),
  since the fleet stack pulls images from it and reuses its CA and client
  cert. The scratch imageserver must trust the `hypervisor-scratch` identity
  (an `-srpcTrustedUsers` entry) — `setup-scratch-imaginator` already adds
  this.
- `ip`, `ebtables`, `openssl`, `qemu-system-x86_64`, `sudo`, and `/dev/kvm`
  present on the host (checked by `setup`/`check_prereqs`).

## Commands

```
setup                First-time setup: check prerequisites, build binaries,
                      mint certs, then start the stack.
start                Start the network and services (after stop or reboot).
stop                 Stop the hypervisor and fleet-manager, and tear down
                      the bridge.
restart              stop followed by start.
status               Show whether the hypervisor and fleet-manager are
                      running.
env                  Print shell exports/aliases; use
                      `eval "$(scripts/setup-scratch-fleet env)"` to load
                      sf-vm (vm-control via the fleet-manager) and sf-hyper
                      (hyper-control against the hypervisor directly).
create-vm stream [name]
                      Create a VM from a scratch imageserver stream, e.g.
                      MyStream/Ubuntu-22/amd64.
destroy-vm ipAddr    Destroy a VM by its IP address.
teardown             stop, then remove the bridge and all state under
                      $SCRATCH_DIR.
```

## Defaults

| Item             | Value             |
|------------------|-------------------|
| Hypervisor port  | 6976              |
| Fleet-manager port | 6977            |
| Imageserver port | 26971             |
| Bridge           | `br@scratch`      |
| Subnet           | 192.168.66.0/24 (gateway 192.168.66.1, pool .10-.59) |

## Caveats

1. **Host-only network, no NAT.** The bridge has no physical uplink and no
   NAT is configured, so VMs are only reachable from the dev VM itself (e.g.
   `nc -z 192.168.66.10 22`). There is no outbound internet access from
   inside a scratch VM.

2. **Guest execution needs working (nested) KVM.** The hypervisor always
   starts QEMU with `accel=kvm` and has no TCG fallback. If the host cannot
   actually execute guest instructions under nested KVM (as was observed on
   the current dev VM, where KVM VM-entry fails), the VM object is created
   and QEMU starts, but the guest never runs any instructions — there is no
   DHCP lease and no SSH. That is a host/hypervisor-nesting limitation, not a
   bug in this script; if `create-vm` reports success but the VM never
   answers on port 22, check nested-KVM support on the host before suspecting
   the script.

Also note: images pulled into the scratch imageserver via a lossy tar+add
copy can be missing `/etc/machine-id`. Pull or build the base layer so that
an empty `/etc/machine-id` file is present in the image (the base image
self-installs a real one on first boot) — see the scratch-imaginator setup
notes for how the seed image is built.

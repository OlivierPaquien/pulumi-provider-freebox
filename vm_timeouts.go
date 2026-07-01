package main

import "time"

// VirtualMachineTimeouts configures operation timeouts for virtual machines.
type VirtualMachineTimeouts struct {
	Create     time.Duration `pulumi:"create,optional"`
	Update     time.Duration `pulumi:"update,optional"`
	Read       time.Duration `pulumi:"read,optional"`
	Delete     time.Duration `pulumi:"delete,optional"`
	Kill       time.Duration `pulumi:"kill,optional"`
	Networking time.Duration `pulumi:"networking,optional"`
}

func defaultVirtualMachineTimeouts() VirtualMachineTimeouts {
	return VirtualMachineTimeouts{
		Create:     5 * time.Minute,
		Update:     5 * time.Minute,
		Read:       5 * time.Minute,
		Delete:     5 * time.Minute,
		Kill:       30 * time.Second,
		Networking: time.Minute,
	}
}

func (t *VirtualMachineTimeouts) resolved() VirtualMachineTimeouts {
	def := defaultVirtualMachineTimeouts()
	if t == nil {
		return def
	}
	out := *t
	if out.Create == 0 {
		out.Create = def.Create
	}
	if out.Update == 0 {
		out.Update = def.Update
	}
	if out.Read == 0 {
		out.Read = def.Read
	}
	if out.Delete == 0 {
		out.Delete = def.Delete
	}
	if out.Kill == 0 {
		out.Kill = def.Kill
	}
	if out.Networking == 0 {
		out.Networking = def.Networking
	}
	return out
}

// VirtualMachinePowerTimeouts configures timeouts for power operations.
type VirtualMachinePowerTimeouts struct {
	Kill time.Duration `pulumi:"kill,optional"`
}

func (t *VirtualMachinePowerTimeouts) killTimeout(fallback int64) time.Duration {
	if t != nil && t.Kill > 0 {
		return t.Kill
	}
	if fallback > 0 {
		return time.Duration(fallback) * time.Second
	}
	return 30 * time.Second
}

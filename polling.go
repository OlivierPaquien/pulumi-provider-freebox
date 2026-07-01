package main

import "time"

// PollingSpec configures interval and timeout for async Freebox tasks.
type PollingSpec struct {
	Interval time.Duration `pulumi:"interval,optional"`
	Timeout  time.Duration `pulumi:"timeout,optional"`
}

func (p *PollingSpec) withDefaults(interval, timeout time.Duration) PollingSpec {
	if p == nil {
		return PollingSpec{Interval: interval, Timeout: timeout}
	}
	out := *p
	if out.Interval == 0 {
		out.Interval = interval
	}
	if out.Timeout == 0 {
		out.Timeout = timeout
	}
	return out
}

// RemoteFilePolling groups polling settings for remote file operations.
type RemoteFilePolling struct {
	Download        *PollingSpec `pulumi:"download,optional"`
	Upload          *PollingSpec `pulumi:"upload,optional"`
	Copy            *PollingSpec `pulumi:"copy,optional"`
	Move            *PollingSpec `pulumi:"move,optional"`
	Delete          *PollingSpec `pulumi:"delete,optional"`
	ChecksumCompute *PollingSpec `pulumi:"checksumCompute,optional"`
	Extract         *PollingSpec `pulumi:"extract,optional"`
}

func defaultRemoteFilePolling() RemoteFilePolling {
	return RemoteFilePolling{
		Download:        &PollingSpec{Interval: 3 * time.Second, Timeout: 30 * time.Minute},
		Upload:          &PollingSpec{Interval: 3 * time.Second, Timeout: 30 * time.Minute},
		Copy:            &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Move:            &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Delete:          &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Extract:         &PollingSpec{Interval: time.Second, Timeout: 2 * time.Minute},
		ChecksumCompute: &PollingSpec{Interval: time.Second, Timeout: 2 * time.Minute},
	}
}

func (p *RemoteFilePolling) resolved() RemoteFilePolling {
	def := defaultRemoteFilePolling()
	if p == nil {
		return def
	}
	out := *p
	if out.Download == nil {
		out.Download = def.Download
	}
	if out.Upload == nil {
		out.Upload = def.Upload
	}
	if out.Copy == nil {
		out.Copy = def.Copy
	}
	if out.Move == nil {
		out.Move = def.Move
	}
	if out.Delete == nil {
		out.Delete = def.Delete
	}
	if out.Extract == nil {
		out.Extract = def.Extract
	}
	if out.ChecksumCompute == nil {
		out.ChecksumCompute = def.ChecksumCompute
	}
	return out
}

// VirtualDiskPolling groups polling settings for virtual disk operations.
type VirtualDiskPolling struct {
	Checksum *PollingSpec `pulumi:"checksum,optional"`
	Copy     *PollingSpec `pulumi:"copy,optional"`
	Create   *PollingSpec `pulumi:"create,optional"`
	Delete   *PollingSpec `pulumi:"delete,optional"`
	Move     *PollingSpec `pulumi:"move,optional"`
	Resize   *PollingSpec `pulumi:"resize,optional"`
}

func defaultVirtualDiskPolling() VirtualDiskPolling {
	return VirtualDiskPolling{
		Checksum: &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Copy:     &PollingSpec{Interval: 2 * time.Second, Timeout: 2 * time.Minute},
		Create:   &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Delete:   &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Move:     &PollingSpec{Interval: time.Second, Timeout: time.Minute},
		Resize:   &PollingSpec{Interval: time.Second, Timeout: time.Minute},
	}
}

func (p *VirtualDiskPolling) resolved() VirtualDiskPolling {
	def := defaultVirtualDiskPolling()
	if p == nil {
		return def
	}
	out := *p
	if out.Checksum == nil {
		out.Checksum = def.Checksum
	}
	if out.Copy == nil {
		out.Copy = def.Copy
	}
	if out.Create == nil {
		out.Create = def.Create
	}
	if out.Delete == nil {
		out.Delete = def.Delete
	}
	if out.Move == nil {
		out.Move = def.Move
	}
	if out.Resize == nil {
		out.Resize = def.Resize
	}
	return out
}

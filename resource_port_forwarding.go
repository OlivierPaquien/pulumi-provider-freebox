package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// PortForwarding resource: manages a port forwarding rule on the Freebox.
type PortForwarding struct{}

type PortForwardingArgs struct {
	Enabled        bool   `pulumi:"enabled"`
	IPProtocol     string `pulumi:"ipProtocol"`
	PortRangeStart int64  `pulumi:"portRangeStart"`
	PortRangeEnd   *int64 `pulumi:"portRangeEnd,optional"`
	TargetPort     *int64 `pulumi:"targetPort,optional"`
	SourceIP       string `pulumi:"sourceIp,optional"`
	TargetIP       string `pulumi:"targetIp"`
	Comment        string `pulumi:"comment,optional"`
}

type PortForwardingState struct {
	PortForwardingArgs
	ID       int64  `pulumi:"id"`
	Hostname string `pulumi:"hostname"`
}

func (PortForwarding) Annotate(a infer.Annotator) {
	a.Describe(&PortForwarding{}, "Manages a port forwarding rule between a local host and the Freebox Internet Gateway.")
	args := &PortForwardingArgs{}
	a.Describe(&args.Enabled, "Whether the forwarding is enabled.")
	a.Describe(&args.IPProtocol, "Protocol: tcp or udp.")
	a.Describe(&args.PortRangeStart, "Start of the WAN port range (inclusive).")
	a.Describe(&args.PortRangeEnd, "End of the WAN port range (defaults to portRangeStart).")
	a.Describe(&args.TargetPort, "Target LAN port (defaults to portRangeStart when range is a single port).")
	a.Describe(&args.SourceIP, "Source IP filter (0.0.0.0 = any).")
	a.Describe(&args.TargetIP, "Local IP of the port forwarding target.")
	a.Describe(&args.Comment, "Optional comment for the rule.")
}

func (PortForwarding) Create(ctx context.Context, req infer.CreateRequest[PortForwardingArgs]) (infer.CreateResponse[PortForwardingState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[PortForwardingState]{}, err
	}

	payload := freeboxTypes.PortForwardingRulePayload{
		Enabled:      &req.Inputs.Enabled,
		IPProtocol:   req.Inputs.IPProtocol,
		LanIP:        req.Inputs.TargetIP,
		WanPortStart: req.Inputs.PortRangeStart,
	}
	if req.Inputs.PortRangeEnd != nil {
		payload.WanPortEnd = *req.Inputs.PortRangeEnd
	} else {
		payload.WanPortEnd = req.Inputs.PortRangeStart
	}
	if req.Inputs.TargetPort != nil {
		payload.LanPort = *req.Inputs.TargetPort
	} else {
		payload.LanPort = req.Inputs.PortRangeStart
	}
	if req.Inputs.SourceIP != "" {
		payload.SourceIP = req.Inputs.SourceIP
	}
	if req.Inputs.Comment != "" {
		payload.Comment = req.Inputs.Comment
	}

	if req.DryRun {
		return infer.CreateResponse[PortForwardingState]{
			ID:     "unknown",
			Output: PortForwardingState{PortForwardingArgs: req.Inputs, Hostname: ""},
		}, nil
	}

	rule, err := cli.CreatePortForwardingRule(ctx, payload)
	if err != nil {
		return infer.CreateResponse[PortForwardingState]{}, fmt.Errorf("create port forwarding: %w", err)
	}

	state := PortForwardingState{
		PortForwardingArgs: req.Inputs,
		ID:                 rule.ID,
		Hostname:           rule.Hostname,
	}
	return infer.CreateResponse[PortForwardingState]{
		ID:     strconv.FormatInt(rule.ID, 10),
		Output: state,
	}, nil
}

func (PortForwarding) Read(ctx context.Context, req infer.ReadRequest[PortForwardingArgs, PortForwardingState]) (infer.ReadResponse[PortForwardingArgs, PortForwardingState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, err
	}

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, fmt.Errorf("invalid port forwarding id: %w", err)
	}

	rule, err := cli.GetPortForwardingRule(ctx, id)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && (apiErr.Code == "not_found" || apiErr.Code == "invalid_request") {
			return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, nil
		}
		return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, fmt.Errorf("get port forwarding: %w", err)
	}

	enabled := false
	if rule.Enabled != nil {
		enabled = *rule.Enabled
	}
	state := PortForwardingState{
		PortForwardingArgs: PortForwardingArgs{
			Enabled:        enabled,
			IPProtocol:     rule.IPProtocol,
			PortRangeStart: rule.WanPortStart,
			PortRangeEnd:   int64Ptr(rule.WanPortEnd),
			TargetPort:     int64Ptr(rule.LanPort),
			SourceIP:       rule.SourceIP,
			TargetIP:       rule.LanIP,
			Comment:        rule.Comment,
		},
		ID:       rule.ID,
		Hostname: rule.Hostname,
	}
	return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{State: state}, nil
}

func (PortForwarding) Update(ctx context.Context, req infer.UpdateRequest[PortForwardingArgs, PortForwardingState]) (infer.UpdateResponse[PortForwardingState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[PortForwardingState]{}, err
	}

	id := req.State.ID
	payload := freeboxTypes.PortForwardingRulePayload{
		Enabled:      &req.Inputs.Enabled,
		IPProtocol:   req.Inputs.IPProtocol,
		LanIP:        req.Inputs.TargetIP,
		WanPortStart: req.Inputs.PortRangeStart,
	}
	if req.Inputs.PortRangeEnd != nil {
		payload.WanPortEnd = *req.Inputs.PortRangeEnd
	} else {
		payload.WanPortEnd = req.Inputs.PortRangeStart
	}
	if req.Inputs.TargetPort != nil {
		payload.LanPort = *req.Inputs.TargetPort
	} else {
		payload.LanPort = req.Inputs.PortRangeStart
	}
	if req.Inputs.SourceIP != "" {
		payload.SourceIP = req.Inputs.SourceIP
	}
	if req.Inputs.Comment != "" {
		payload.Comment = req.Inputs.Comment
	}

	if req.DryRun {
		return infer.UpdateResponse[PortForwardingState]{
			Output: PortForwardingState{PortForwardingArgs: req.Inputs, ID: id, Hostname: req.State.Hostname},
		}, nil
	}

	rule, err := cli.UpdatePortForwardingRule(ctx, id, payload)
	if err != nil {
		return infer.UpdateResponse[PortForwardingState]{}, fmt.Errorf("update port forwarding: %w", err)
	}

	state := PortForwardingState{
		PortForwardingArgs: req.Inputs,
		ID:                 rule.ID,
		Hostname:           rule.Hostname,
	}
	return infer.UpdateResponse[PortForwardingState]{Output: state}, nil
}

func (PortForwarding) Delete(ctx context.Context, req infer.DeleteRequest[PortForwardingState]) error {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return err
	}
	id := req.State.ID
	return cli.DeletePortForwardingRule(ctx, id)
}

func int64Ptr(i int64) *int64 { return &i }

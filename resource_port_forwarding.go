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
	ID       int64    `pulumi:"ruleId"` // id réservé par Pulumi
	Hostname string   `pulumi:"hostname"`
	Host     *LanHost `pulumi:"host,optional"`
}

func (args *PortForwardingArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Enabled, "Whether the forwarding is enabled.")
	a.Describe(&args.IPProtocol, "Protocol: tcp or udp.")
	a.Describe(&args.PortRangeStart, "Start of the WAN port range (inclusive).")
	a.Describe(&args.PortRangeEnd, "End of the WAN port range (defaults to portRangeStart).")
	a.Describe(&args.TargetPort, "Target LAN port (defaults to portRangeStart when range is a single port).")
	a.Describe(&args.SourceIP, "Source IP filter (0.0.0.0 = any).")
	a.Describe(&args.TargetIP, "Local IP of the port forwarding target.")
	a.Describe(&args.Comment, "Optional comment for the rule.")
}

func (st *PortForwardingState) Annotate(a infer.Annotator) {
	a.Describe(&st.ID, "Freebox API rule ID.")
	a.Describe(&st.Hostname, "Hostname reported by the Freebox for this rule.")
	a.Describe(&st.Host, "LAN host information for the target.")
}

func (PortForwarding) Annotate(a infer.Annotator) {
	a.SetToken("fw", "PortForwarding")
	// PortForwarding est une struct vide ; l'Annotator reçoit ce type uniquement.
	// Tout Describe (ressource, args ou state) provoque "reflect.Value.Addr of unaddressable value".
}

func portForwardingPayload(args PortForwardingArgs) freeboxTypes.PortForwardingRulePayload {
	payload := freeboxTypes.PortForwardingRulePayload{
		Enabled:      &args.Enabled,
		IPProtocol:   args.IPProtocol,
		LanIP:        args.TargetIP,
		WanPortStart: args.PortRangeStart,
	}
	if args.PortRangeEnd != nil {
		payload.WanPortEnd = *args.PortRangeEnd
	} else {
		payload.WanPortEnd = args.PortRangeStart
	}
	if args.TargetPort != nil {
		payload.LanPort = *args.TargetPort
	} else {
		payload.LanPort = args.PortRangeStart
	}
	if args.SourceIP != "" {
		payload.SourceIP = args.SourceIP
	} else {
		payload.SourceIP = "0.0.0.0"
	}
	if args.Comment != "" {
		payload.Comment = args.Comment
	}
	return payload
}

func portForwardingStateFromRule(args PortForwardingArgs, rule freeboxTypes.PortForwardingRule) PortForwardingState {
	return PortForwardingState{
		PortForwardingArgs: args,
		ID:                 rule.ID,
		Hostname:           rule.Hostname,
		Host:               lanHostPtrFromAPI(rule.Host),
	}
}

func portForwardingArgsFromRule(rule freeboxTypes.PortForwardingRule) PortForwardingArgs {
	enabled := false
	if rule.Enabled != nil {
		enabled = *rule.Enabled
	}
	return PortForwardingArgs{
		Enabled:        enabled,
		IPProtocol:     rule.IPProtocol,
		PortRangeStart: rule.WanPortStart,
		PortRangeEnd:   int64Ptr(rule.WanPortEnd),
		TargetPort:     int64Ptr(rule.LanPort),
		SourceIP:       rule.SourceIP,
		TargetIP:       rule.LanIP,
		Comment:        rule.Comment,
	}
}

func (PortForwarding) Create(ctx context.Context, req infer.CreateRequest[PortForwardingArgs]) (infer.CreateResponse[PortForwardingState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[PortForwardingState]{}, err
	}

	payload := portForwardingPayload(req.Inputs)

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

	state := portForwardingStateFromRule(req.Inputs, rule)
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
		if errors.Is(err, client.ErrPortForwardingRuleNotFound) {
			return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, nil
		}
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && (apiErr.Code == "not_found" || apiErr.Code == "invalid_request" || apiErr.Code == "noent") {
			return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, nil
		}
		return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{}, fmt.Errorf("get port forwarding: %w", err)
	}

	state := portForwardingStateFromRule(portForwardingArgsFromRule(rule), rule)
	return infer.ReadResponse[PortForwardingArgs, PortForwardingState]{State: state}, nil
}

func (PortForwarding) Update(ctx context.Context, req infer.UpdateRequest[PortForwardingArgs, PortForwardingState]) (infer.UpdateResponse[PortForwardingState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[PortForwardingState]{}, err
	}

	id := req.State.ID
	payload := portForwardingPayload(req.Inputs)

	if req.DryRun {
		return infer.UpdateResponse[PortForwardingState]{
			Output: PortForwardingState{PortForwardingArgs: req.Inputs, ID: id, Hostname: req.State.Hostname},
		}, nil
	}

	rule, err := cli.UpdatePortForwardingRule(ctx, id, payload)
	if err != nil {
		return infer.UpdateResponse[PortForwardingState]{}, fmt.Errorf("update port forwarding: %w", err)
	}

	state := portForwardingStateFromRule(req.Inputs, rule)
	return infer.UpdateResponse[PortForwardingState]{Output: state}, nil
}

func (PortForwarding) Delete(ctx context.Context, req infer.DeleteRequest[PortForwardingState]) (infer.DeleteResponse, error) {
	freeboxLog("[freebox] PortForwarding Delete: ruleId=%d\n", req.State.ID)
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	id := req.State.ID
	err = cli.DeletePortForwardingRule(ctx, id)
	if err != nil {
		freeboxLog("[freebox] PortForwarding Delete ruleId=%d: %v\n", req.State.ID, err)
		return infer.DeleteResponse{}, err
	}
	freeboxLog("[freebox] PortForwarding Delete ruleId=%d: success\n", req.State.ID)
	return infer.DeleteResponse{}, nil
}

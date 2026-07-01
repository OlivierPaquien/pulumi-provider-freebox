// Integration tests for freebox:fw:PortForwarding (same spirit as Terraform resource_port_forwarding_test.go).
// Require FREEBOX_APP_ID and FREEBOX_TOKEN.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/nikolalohinski/free-go/client"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const portForwardingType = "freebox:fw:PortForwarding"

func portForwardingURN(name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(portForwardingType), name)
}

func testPortForwarding(t *testing.T) {
	ctx := context.Background()
	cfg := freeboxTestConfig()
	server := newTestServer(t, cfg)
	freeboxClient := newFreeboxClient(t, ctx)

	// Random values to avoid clashes with existing rules
	rng := rand.New(rand.NewSource(0))
	sourceSubnet := rng.Int63n(200-2) + 2
	sourceIP := fmt.Sprintf("1.1.1.%d", sourceSubnet)
	targetIP := fmt.Sprintf("192.168.1.%d", (sourceSubnet%200)+1)
	portRangeStart := rng.Int63n(60000) + 1024
	portRangeEnd := portRangeStart + 10
	targetPort := portRangeStart
	resourceName := "test-pf-" + strconv.FormatInt(portRangeStart, 10)
	comment := "pulumi-provider-freebox-test"

	t.Run("create and delete", func(t *testing.T) {
		urn := portForwardingURN(resourceName)
		props := property.NewMap(map[string]property.Value{
			"enabled":        property.New(true),
			"ipProtocol":     property.New("tcp"),
			"comment":        property.New(comment),
			"sourceIp":       property.New(sourceIP),
			"portRangeStart": property.New(float64(portRangeStart)),
			"portRangeEnd":   property.New(float64(portRangeEnd)),
			"targetIp":       property.New(targetIP),
			"targetPort":     property.New(float64(targetPort)),
		})

		// Create
		createResp, err := server.Create(p.CreateRequest{Urn: urn, Properties: props})
		require.NoError(t, err)
		require.NotEmpty(t, createResp.ID)

		ruleID, err := strconv.ParseInt(createResp.ID, 10, 64)
		require.NoError(t, err)

		// Read back via API
		rule, err := freeboxClient.GetPortForwardingRule(ctx, ruleID)
		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, true, *rule.Enabled)
		assert.Equal(t, "tcp", rule.IPProtocol)
		assert.Equal(t, comment, rule.Comment)
		assert.Equal(t, sourceIP, rule.SourceIP)
		assert.Equal(t, portRangeStart, rule.WanPortStart)
		assert.Equal(t, portRangeEnd, rule.WanPortEnd)
		assert.Equal(t, targetPort, rule.LanPort)
		assert.Equal(t, targetIP, rule.LanIP)

		// Delete via provider
		err = server.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn, Properties: createResp.Properties})
		require.NoError(t, err)

		// CheckDestroy: rule must be gone on the Freebox
		_, err = freeboxClient.GetPortForwardingRule(ctx, ruleID)
		assert.ErrorIs(t, err, client.ErrPortForwardingRuleNotFound)
	})

	t.Run("create, update and delete", func(t *testing.T) {
		urn := portForwardingURN(resourceName + "-up")
		props := property.NewMap(map[string]property.Value{
			"enabled":        property.New(true),
			"ipProtocol":     property.New("tcp"),
			"comment":        property.New(comment + "-up"),
			"sourceIp":       property.New(sourceIP),
			"portRangeStart": property.New(float64(portRangeStart + 100)),
			"portRangeEnd":   property.New(float64(portRangeStart + 100)),
			"targetIp":       property.New(targetIP),
			"targetPort":     property.New(float64(portRangeStart + 100)),
		})

		createResp, err := server.Create(p.CreateRequest{Urn: urn, Properties: props})
		require.NoError(t, err)
		require.NotEmpty(t, createResp.ID)

		// Update: change port range and target port
		newStart := portRangeStart + 200
		newTargetPort := portRangeStart + 201
		newInputs := property.NewMap(map[string]property.Value{
			"enabled":        property.New(true),
			"ipProtocol":     property.New("tcp"),
			"comment":        property.New(comment + "-up"),
			"sourceIp":       property.New(sourceIP),
			"portRangeStart": property.New(float64(newStart)),
			"portRangeEnd":   property.New(float64(newStart)),
			"targetIp":       property.New(targetIP),
			"targetPort":     property.New(float64(newTargetPort)),
		})

		updateResp, err := server.Update(p.UpdateRequest{
			ID:        createResp.ID,
			Urn:       urn,
			State:     createResp.Properties,
			Inputs:    newInputs,
			OldInputs: createResp.Properties,
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)

		ruleID, _ := strconv.ParseInt(createResp.ID, 10, 64)
		rule, err := freeboxClient.GetPortForwardingRule(ctx, ruleID)
		require.NoError(t, err)
		assert.Equal(t, newStart, rule.WanPortStart)
		assert.Equal(t, newStart, rule.WanPortEnd)
		assert.Equal(t, newTargetPort, rule.LanPort)

		err = server.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn, Properties: updateResp.Properties})
		require.NoError(t, err)
		_, err = freeboxClient.GetPortForwardingRule(ctx, ruleID)
		assert.ErrorIs(t, err, client.ErrPortForwardingRuleNotFound)
	})
}

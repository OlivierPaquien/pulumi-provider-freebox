// Tests unitaires du provider (mock HTTP, pas de Freebox réelle).
// Inspirés de terraform-provider-freebox/internal/provider_test.go.
package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getApiVersionToken : token public (ModuleMap expose le module "index").
const getApiVersionToken = "freebox:api:Version"

// apiVersionJSON est le corps de réponse mock pour GET /api/v0/api_version (ou /api/v42/api_version).
func apiVersionJSON() string {
	return `{
		"box_model_name": "Freebox v7 (r1)",
		"api_base_url": "/api/",
		"https_port": 12345,
		"device_name": "Freebox Server",
		"https_available": true,
		"box_model": "fbxgw7-r1/full",
		"api_domain": "xxxxxxxx.fbxos.fr",
		"uid": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"api_version": "11.1",
		"device_type": "FreeboxServer7,1"
	}`
}

func TestProvider_InvalidEndpoint(t *testing.T) {
	// Endpoint invalide : client.New doit échouer (ou la requête échoue).
	// free-go renvoie "can not build base url from endpoint" quand url.Parse échoue.
	cfg := Config{
		Endpoint:   " \t\n", // caractères qui peuvent faire échouer url.Parse selon le contexte
		APIVersion: "v0",
		AppID:      "test",
		Token:      "x",
	}
	server := newTestServer(t, cfg)
	_, err := server.Invoke(p.InvokeRequest{
		Token: tokens.Type(getApiVersionToken),
		Args:  property.Map{},
	})
	require.Error(t, err)
	errMsg := err.Error()
	assert.Regexp(t, regexp.MustCompile("can not build base url from endpoint|failed to build request|invalid"), errMsg, "error should mention invalid endpoint or request")
}

func TestProvider_ConfigFromEnv(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_version" || r.URL.Path == "/api/v0/api_version" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(apiVersionJSON()))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mock.Close()

	// Config via champs explicites (équivalent env)
	cfg := Config{
		Endpoint:   mock.URL,
		APIVersion: "v0",
		AppID:      "test",
		Token:      "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	}
	srv := newTestServer(t, cfg)
	resp, err := srv.Invoke(p.InvokeRequest{
		Token: tokens.Type(getApiVersionToken),
		Args:  property.Map{},
	})
	require.NoError(t, err)
	require.Empty(t, resp.Failures)

	assert.Equal(t, property.New("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"), resp.Return.Get("uid"))
	assert.Equal(t, property.New("11.1"), resp.Return.Get("apiVersion"))
	assert.Equal(t, property.New(float64(12345)), resp.Return.Get("httpsPort"))
	assert.Equal(t, property.New("Freebox v7 (r1)"), resp.Return.Get("boxModelName"))
	assert.Equal(t, property.New(true), resp.Return.Get("httpsAvailable"))
}

func TestProvider_ConfigFromBlock(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// free-go appelle base + "/api_version", base = endpoint + "/api/" + version
		if r.URL.Path == "/api_version" || r.URL.Path == "/api/v42/api_version" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(apiVersionJSON()))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mock.Close()

	cfg := Config{
		Endpoint:   mock.URL,
		APIVersion: "v42",
		AppID:      "test",
		Token:      "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	}
	srv := newTestServer(t, cfg)
	resp, err := srv.Invoke(p.InvokeRequest{
		Token: tokens.Type(getApiVersionToken),
		Args:  property.Map{},
	})
	require.NoError(t, err)
	require.Empty(t, resp.Failures)
	assert.Equal(t, property.New("11.1"), resp.Return.Get("apiVersion"))
}

const dhcpStaticLeaseType = "freebox:dhcp:StaticLease"
const getDhcpLeaseToken = "freebox:dhcp:getLease"

func TestProvider_DHCPStaticLeaseInvalidMAC(t *testing.T) {
	cfg := Config{
		Endpoint:   "http://localhost",
		APIVersion: "v0",
		AppID:      "test",
		Token:      "x",
	}
	server := newTestServer(t, cfg)
	urn := resource.NewURN("stack", "proj", "", tokens.Type(dhcpStaticLeaseType), "bad-lease")

	_, err := server.Create(p.CreateRequest{
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"mac": property.New("not-a-mac"),
			"ip":  property.New("192.168.1.10"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mac")
}

func TestProvider_GetDhcpLeaseInvalidMAC(t *testing.T) {
	cfg := Config{
		Endpoint:   "http://localhost",
		APIVersion: "v0",
		AppID:      "test",
		Token:      "x",
	}
	server := newTestServer(t, cfg)

	_, err := server.Invoke(p.InvokeRequest{
		Token: tokens.Type(getDhcpLeaseToken),
		Args: property.NewMap(map[string]property.Value{
			"mac": property.New("invalid"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mac")
}

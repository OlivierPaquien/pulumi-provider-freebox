package main

import "github.com/pulumi/pulumi-go-provider/infer"

// Config holds the Freebox provider configuration.
type Config struct {
	Endpoint   string `pulumi:"endpoint,optional"`
	APIVersion string `pulumi:"apiVersion,optional"`
	AppID      string `pulumi:"appId" provider:"secret"`
	Token      string `pulumi:"token" provider:"secret"`
}

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(c, "Configuration for the Freebox provider.")
	a.Describe(&c.Endpoint, "The address of the Freebox (default: http://mafreebox.freebox.fr).")
	a.Describe(&c.APIVersion, "The version of the API to use (default: latest).")
	a.Describe(&c.AppID, "The ID of the application for Freebox API authentication.")
	a.Describe(&c.Token, "The private token for Freebox API authentication.")
}

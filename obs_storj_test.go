package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"storj.io/uplink/edge"
)

func TestStorjUplink(t *testing.T) {
	client, err := newObsStorjClient(context.Background(), obsStorjOptions{
		// Docs: https://docs.storj.io/dcs/getting-started/quickstart-uplink-cli/generate-access-grants-and-tokens/generate-a-token
		// SatelliteAddress: "link.storjshare.io",
		// APIKey:           "xxx",
		// Passphrase:       "xxxx",
		// --- or ---
		// Docs: https://docs.storj.io/dcs/concepts/access/access-grants
		// AccessGrant: "xxxx",
		// --- or ---
		// Docs: https://pkg.go.dev/storj.io/uplink/edge#Config.RegisterAccess
		// > RegisterAccess gets credentials for the Storj-hosted Gateway and linkshare service.
		// > All files accessible under the Access are then also accessible via those services.
		// > If you call this function a lot, and the use case allows it, please limit
		// > the lifetime of the credentials by setting Permission.NotAfter when creating the Access.
		AccessKeyID: "placeholder",
	})
	require.NoError(t, err)

	if client.access != nil {
		println(client.access.SatelliteAddress())
	}

	shareLinkURL, err := client.JoinShareURL(
		// "demo-bucket", "main.c",
		"moe", "moe-onl/13744453-430b-4e6b-ae81-29e7f2491317.png",
		&edge.ShareURLOptions{
			Raw: true,
		})
	require.NoError(t, err)
	println(shareLinkURL)
}

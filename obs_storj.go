package main

import (
	"context"
	"flag"
	"os"

	"github.com/pkg/errors"
	"storj.io/uplink"
	"storj.io/uplink/edge"
)

type obsStorjOptions struct {
	SatelliteAddress string
	APIKey           string
	Passphrase       string

	AccessGrant string

	AccessKeyID  string
	ShareBaseURL string
}

var defaultObsUplinkOpts = obsStorjOptions{
	// we can override satellite address from Access Grant with this
	// ex. "ap1.storj.io:7777"
	SatelliteAddress: "",
	ShareBaseURL:     "https://link.storjshare.io",
}

func (opts *obsStorjOptions) Bind(fs *flag.FlagSet) (err error) {
	var vSatelliteAddr = opts.SatelliteAddress
	if sSatelliteAddr := os.Getenv("UPLINK_SATELLITE_ADDR"); sSatelliteAddr != "" {
		vSatelliteAddr = sSatelliteAddr
	}
	fs.StringVar(&opts.SatelliteAddress, "uplink-satellite-addr", vSatelliteAddr, "OBS Storj Satellite Address")

	{
		var vAPIKey = opts.APIKey
		if sAPIKey := os.Getenv("UPLINK_API_KEY"); sAPIKey != "" {
			vAPIKey = sAPIKey
		}
		fs.StringVar(&opts.APIKey, "uplink-api-key", vAPIKey, "OBS Storj API key")

		var vPassphrase = opts.Passphrase
		if sPassphrase := os.Getenv("UPLINK_PASSPHRASE"); sPassphrase != "" {
			vPassphrase = sPassphrase
		}
		fs.StringVar(&opts.Passphrase, "uplink-passphrase", vPassphrase, "OBS Storj Passphrase")
	}

	var vAccessGrant = opts.AccessGrant
	if sAccessGrant := os.Getenv("UPLINK_ACCESS_GRANT"); sAccessGrant != "" {
		vAccessGrant = sAccessGrant
	}
	fs.StringVar(&opts.AccessGrant, "uplink-access-grant", vAccessGrant, "OBS Storj Access Grant")

	var vAccessKeyID = opts.AccessKeyID
	if sAccessKeyID := os.Getenv("UPLINK_ACCESS_KEY_ID"); sAccessKeyID != "" {
		vAccessKeyID = sAccessKeyID
	}
	fs.StringVar(&opts.AccessKeyID, "uplink-access-key-id", vAccessKeyID, "OBS Storj Access key ID")

	var vShareBaseURL = opts.ShareBaseURL
	if sShareBaseURL := os.Getenv("UPLINK_SHARE_BASE_URL"); sShareBaseURL != "" {
		vShareBaseURL = sShareBaseURL
	}
	fs.StringVar(&opts.ShareBaseURL, "uplink-share-base-url", vShareBaseURL, "OBS Storj Link Share base URL")

	return
}

var defaultEdgeConfig = edge.Config{
	AuthServiceAddress: "auth.storjshare.io:7777",
}

type storjAggegrateClient struct {
	edgeConfig *edge.Config
	access     *uplink.Access
	project    *uplink.Project

	creds *edge.Credentials

	accessKeyID  string
	shareBaseURL string
}

func (c *storjAggegrateClient) Init(ctx context.Context) (_ *storjAggegrateClient, err error) {
	if c.access != nil {
		// TODO: rate limit this, persist the registered access key ID
		if c.creds, err = c.edgeConfig.RegisterAccess(ctx, c.access, &edge.RegisterAccessOptions{
			// This will create `accessKeyID` that allow anonymous access
			// to any objects that `c.access` has access to (just like `public-read` ACL).
			Public: true,
		}); err != nil {
			err = errors.Wrap(err, "register access")
			return
		}
	}
	return c, nil
}

func (c *storjAggegrateClient) getAccessKeyID() (accessKeyID string) {
	accessKeyID = c.accessKeyID
	// fallback to `c.creds` if custom accessKeyID is not provided
	// and `c.creds` is specified.
	if accessKeyID == "" && c.creds != nil {
		accessKeyID = c.creds.AccessKeyID
	}
	return
}

func (c *storjAggegrateClient) getProject() *uplink.Project {
	return c.project
}

func (c *storjAggegrateClient) JoinShareURL(bucket, key string, opts *edge.ShareURLOptions) (string, error) {
	accessKeyID := c.getAccessKeyID()
	return edge.JoinShareURL(c.shareBaseURL, accessKeyID, bucket, key, opts)
}

func newObsStorjClient(ctx context.Context, opts obsStorjOptions) (client *storjAggegrateClient, err error) {
	var (
		access  *uplink.Access
		project *uplink.Project
	)
	switch {
	case opts.AccessGrant != "":
		// Docs: https://docs.storj.io/dcs/concepts/access/access-grants
		access, err = uplink.ParseAccess(opts.AccessGrant)
		if err != nil {
			err = errors.Wrap(err, "parse access")
			return
		}
	case opts.APIKey != "" && opts.Passphrase != "":
		// Docs: https://docs.storj.io/dcs/getting-started/quickstart-uplink-cli/generate-access-grants-and-tokens/generate-a-token
		if access, err = uplink.RequestAccessWithPassphrase(
			ctx,
			opts.SatelliteAddress,
			opts.APIKey,
			opts.Passphrase,
		); err != nil {
			err = errors.Wrap(err, "access with passphrase")
			return
		}
	default:
		// using custom accessKeyID
		access = nil
	}
	// TODO: limit access scope.
	if access != nil {
		// Consider to limit the access to specific bucket and prefix
		// access, err = access.Share(
		// 	uplink.ReadOnlyPermission(),
		// 	uplink.SharePrefix{
		// 		Bucket: "",
		// 		Prefix: "",
		// 	})

		// open project
		project, err = uplink.OpenProject(ctx, access)
		if err != nil {
			err = errors.Wrap(err, "open project")
			return
		}
	}
	// fallback default link sharing base url
	if opts.ShareBaseURL == "" {
		opts.ShareBaseURL = defaultObsUplinkOpts.ShareBaseURL
	}
	return (&storjAggegrateClient{
		edgeConfig: &defaultEdgeConfig,
		access:     access,
		project:    project,

		accessKeyID:  opts.AccessKeyID,
		shareBaseURL: opts.ShareBaseURL,
	}).Init(ctx)
}

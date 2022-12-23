# obs-access-signer

S3 Object Storage access signer.

Run `obs-access-signer` behind a gateway/cache proxy is preferred as the response is static.

There's an example of using it with Varnish Cache, which you can see [here](docker/docker-compose.yaml).

## Why?

Some S3-compatible gateways might not support ACL endpoints but they support presigned access. Currently, the behavior of `obs-access-signer` is similar to `public-read` ACL where clients can access objects anonymously and redirect them (permanently) to presigned URL with `Expires` set to the max signed value of `int64` which has roughly 250yrs lifetime since UNIX time started.

Update: starting at v0.0.3, obs-access-signer supports custom redirection status code and expiry.

## License

Apache-2.0

# obs-access-signer

S3 Object Storage access signer.

Run `obs-access-signer` behind a gateway/cache proxy is preferred as the response is static.

There's an example of using it with Varnish Cache, you can see [here](docker/docker-compose.yaml).

## Why?

Some S3-compatible gateway might not support ACL endpoints but they are support presigned access. Currently, the behavior of `obs-access-signer` is similar to `public-read` ACL where clients can access objects anonymously and redirect them (permanently) to presigned url with `Expires` set to the max signed value of `int64` which has roughly 250yrs lifetime since unix time started.


## License

Apache-2.0

#
# This is an example VCL file for Varnish.
#
# It does not do anything by default, delegating control to the
# builtin VCL. The builtin VCL is called when there is no explicit
# return statement.
#
# See the VCL chapters in the Users Guide at https://www.varnish-cache.org/docs/
# and https://www.varnish-cache.org/trac/wiki/VCLExamples for more examples.

# Marker to tell the VCL compiler that this VCL has been adapted to the
# new 4.0 format.
vcl 4.0;

# Default backend definition. Set this to point to your content server.
backend default {
    .host = "obs-access-signer";
    .port = "9002";
    .connect_timeout = 300s;
    .first_byte_timeout = 300s;
    .between_bytes_timeout = 300s;
    .max_connections = 800;
}

sub vcl_hash {
    hash_data(req.url);
    if (req.http.Host) {
        hash_data(req.http.Host);
    } else {
        hash_data(server.ip);
    }
}

sub vcl_recv {
    # Happens before we check if we have this in cache already.
    #
    # Typically you clean up the request here, removing cookies you don't need,
    # rewriting the request, etc.
    
    if (req.method == "PRI") {
        /* We do not support SPDY or HTTP/2.0 */
        return (synth(405));
    }

    # remove port from Host
    set req.http.Host = regsub(req.http.Host, ":[0-9]+", "");

    /* Backend accept HEAD and GET only */
    if (req.method != "GET" && req.method != "HEAD") {
        return (synth(405));
    }

    # Ignore the query string
    set req.url = regsub(req.url, "\?.*$", "");

    return (hash);
}

sub vcl_backend_response {
    # Happens after we have read the response headers from the backend.
    #
    # Here you clean the response headers, removing silly Set-Cookie headers
    # and other mistakes your backend does.

    # Don't cache 400s
    if (beresp.status >= 400) {
        set beresp.uncacheable = true;
        set beresp.http.X-Cacheable = "NO: beresp.status";
        set beresp.ttl = 0s;
        return (deliver);
    }

    # keep last content in case backend goes down.
    set beresp.grace = 6h;

    # cache timeout
    set beresp.ttl = 1h;

    return (deliver);
}

sub vcl_deliver {
    # Happens when we have all the pieces we need, and are about to send the
    # response to the client.
    #
    # You can do accounting or modifying the final object here.

    set resp.http.Via = regsuball(resp.http.Via, "\s\([a-zA-Z0-9\/.]+\)", "");
    set resp.http.Server = "VOAS";

    # Debug header to see if it's a HIT/MISS and the number of hits
    if (obj.hits > 0) {
        set resp.http.X-Cache = "HIT";
    } else {
        set resp.http.X-Cache = "MISS";
    }

    # Please note that obj.hits behaviour changed in 4.0, now it counts per objecthead, not per object
    # and obj.hits may not be reset in some cases where bans are in use. See bug 1492 for details.
    # So take hits with a grain of salt
    set resp.http.X-Cache-Hits = obj.hits;

    unset resp.http.Date;
    unset resp.http.Age;
    # unset resp.http.Server;
    # unset resp.http.Via;

    return (deliver);
}

sub vcl_backend_error {
    return (retry);
}
# Flounder

A lightweight server to help users build simple Gemini sites over http(s)

Designed to help make the Gemini ecosystem more accessible.


## Hosting

Flounder is designed to be very simple to host, and should be able to be relatively easily run by a single person.

Once you've installed Flounder, you'll want to set the configuration variables. The `flounder.toml` file in this directory provides some example configuration.

1. Install with `go get ...`
2. For local testing, flounder will generate a TLS cert for you. However, for production, you'll need to generate a cert that matches \*.your-domain signed by a Certificate Authority.
3. Set the cookie store key

Flounder uses the HTTP templates in the specified templates folder. If you want to modify the look and feel of your site, or host new files, you can modify these files.

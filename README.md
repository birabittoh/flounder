# Flounder: a portal into the small web 

A lightweight server to help users build simple Gemini sites over http(s) and serve those sites over http(s) and Gemini

Flounder is in ALPHA -- development and features are changing frequently, especially as the Gemini spec and ecosystem remains relatively unstable.

See the flagship instance at https://flounder.online and gemini://flounder.online

## Building
Requirements:
go >= 1.15
sqlite dev libraries

To run locally, copy example-config.toml to flounder.toml, then run:

`go run . serve`

Add the following to `/etc/hosts` (include any other users you want to create):

```
127.0.0.1 flounder.local admin.flounder.local proxy.flounder.local
```

## TLS Certs and Reverse Proxy

Gemini TLS certs are handled for you. For HTTP, you'll need a reverse proxy that does the following for you:

1. Cert for yourdomain.whatever
1. Wildcard cert for \*.yourdomain.whatever
2. On Demand cert for custom user domains

If you have a very small deployment of say, <100 users, for example, you can use on demand cert for all domains, and you can skip step 2.

However, for a larger deployment, you'll have to set up a wildcard cert. Wildcard certs are a bit of a pain and difficult to do automatically, depending on your DNS provider. For information on doing this via Caddy, you can follow this guide: https://caddy.community/t/how-to-use-dns-provider-modules-in-caddy-2/8148. 

For information on using certbot to manage wildcard certs, see this guide: https://letsencrypt.org/docs/challenge-types/#dns-01-challenge

An example simple Caddyfile using on-demand certs is available in Caddyfile.example

If you want to host using a server other than Caddy, there's no reason you can't, but it may be more cumbersome to setup the http certs.

You can run this locally by removing the port from the Host config

## Development

Patches are welcome!

* [Ticket tracker](https://todo.sr.ht/~alexwennerberg/flounder)
* [Mailing list](https://lists.sr.ht/~alexwennerberg/flounder-discuss)

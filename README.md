# Flounder: a portal into the small web

A lightweight server to help users build simple Gemini sites over http(s)

Designed to help make the Gemini ecosystem more accessible.

## Building

Requires go 1.15 and sqlite3 development libraries.

`go build`

## Running

For local development, generate some ssl certs, copy `example-config.toml` to `flounder.toml` and set the env variables, then run:

`./flounder serve`

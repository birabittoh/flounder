# Flounder: a portal into the small web

A lightweight server to help users build simple Gemini sites over http(s)

Designed to help make the Gemini ecosystem more accessible.

## Building

Requires go 1.15 and sqlite3 development libraries.

`go build`

## Running

For the Http(s) server, you have two options:

1. Expose the flounder server to the internet directly

2. Route the flounder server through a reverse proxy

This will determine how you set the values in flounder.toml

(More TBD)

`./flounder serve`

## Admin

Flounder comes with an admin panel acessible to users with the admin db flag set.

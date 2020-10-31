# Flounder: a portal into the small web

A lightweight server to help users build simple Gemini sites over http(s) and serve those sites over http(s) and gemini

## Building

Requires go 1.15 and sqlite3 development libraries.

`go build`

To run locally, you can use the example-config.toml to start a test server. Easy!

`./flounder serve`

## Deploying

For the Http(s) server, you have two options:

1. Expose the flounder server to the internet directly

2. Route the flounder server through a reverse proxy

This will determine how you set the values in flounder.toml 


## Admin

Backup your users' data regularly! The admin deletion commands are irreversible.

Flounder comes with an admin panel acessible to users with the admin db flag set. 

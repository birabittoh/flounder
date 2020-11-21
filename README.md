# Flounder: a portal into the small web 

A lightweight server to help users build simple Gemini sites over http(s) and serve those sites over http(s) and Gemini

Flounder is in ALPHA -- development and features are changing frequently, especially as the Gemini spec and ecosystem remains relatively unstable.

## Building

Requires go >= 1.15 and sqlite3 development libraries.

`go build`

To run locally, you can use the example-config.toml to start a test server. Easy!

`./flounder -c example-config.toml serve`

## Deploying

For the Http(s) server, you have two options:

1. Expose the flounder server to the internet directly

2. Route the flounder server through a reverse proxy

This will determine how you set the values in flounder.toml 


## Admin

To make yourself an admin, create a user through the web browser, then run `./flounder -c [config_path] make-admin [your user]` -- this gives you access to admin tools to manage users.

Backup your users' data regularly! The admin deletion commands are irreversible.

Flounder comes with an admin panel accessible to users with the admin db flag set. 

You can also manage users directly through the sqlite database.

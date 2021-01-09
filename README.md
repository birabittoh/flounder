# Flounder: a portal into the small web 

A lightweight server to help users build simple Gemini sites over http(s) and serve those sites over http(s) and Gemini

Flounder is in ALPHA -- development and features are changing frequently, especially as the Gemini spec and ecosystem remains relatively unstable.

See the flagship instance at https://flounder.online and gemini://flounder.online

## Building

Requires go >= 1.15 and sqlite3 development libraries.

`go build`

To run locally, you can use the example-config.toml to start a test server. 

`./flounder -c example-config.toml serve`

After creating an account, you can activate it at the command line with:

./flounder -c example-config.toml activate-user [your-username]

## Deploying

For production, in order to handle TLS and custom domains, I use Caddy with On Demand TLS certs.

(TODO -- rewrite)

I have not extensively tested the self-hosting capabilities, but making it easy to self-host Flounder for either a single or multi-user instance is a goal of mine. Email me if you encounter issues or would like guidance.

## Admin

To make yourself an admin, create a user through the web browser, then run `./flounder -c [config_path] make-admin [your user]` -- this gives you access to admin tools to manage users.

Backup your users' data regularly! The admin deletion commands are irreversible.

Flounder comes with an admin panel accessible to users with the admin db flag set. Admins can also impersonate users if you need to take actions like moderating their content, deleting their account, changing their password, etc.

## Development

Patches are welcome!

* [Ticket tracker](https://todo.sr.ht/~alexwennerberg/flounder)
* [Mailing list](https://lists.sr.ht/~alexwennerberg/flounder-discuss)

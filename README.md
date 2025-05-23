![Baton Logo](./docs/images/baton-logo.png)

# `baton-freshservice` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-freshservice.svg)](https://pkg.go.dev/github.com/conductorone/baton-freshservice) ![main ci](https://github.com/conductorone/baton-freshservice/actions/workflows/main.yaml/badge.svg)

`baton-freshservice` is a connector for [Freshservice](https://www.freshworks.com/freshservice/) built using the [Baton SDK](https://github.com/conductorone/baton-sdk).

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

Freshservice is an online IT service with a fresh twist. When you sign up for Freshservice, you will be offered a 14-day free trial will all the features. Go to [https://www.freshworks.com/freshservice/signup/](https://www.freshworks.com/freshservice/signup/). You can either sign up using your existing Google account, or create a new account by filling the details mentioned in the sign up form. Once you’re done with filling all the details, click on `Try it Free`. 

## Prerequisites

API key and domain for your Freshworks account. If you don't already have one follow the steps [here](https://support.freshservice.com/support/solutions/articles/232987-setting-up-your-freshservice-account) to create a fresh service account and get your domain and api key. 

Your domain name is the subdomain provided by freshservice. 
Ex: https://domain.freshservice.com

https://support.freshservice.com -> support

https://solutions.freshservice.com -> solutions

## How to get your Freshservice API key (3 steps) 

1.- Log in, then, on the top right corner of Freshservice's homepage, you should see an icon of a person.

2.- Click on profile settings.

3.- Complete the CAPTCHA to access your API key.

NOTE: if you can't see your API Key there, you should enable it for your user. For that:
1.- While using an admin account, go to Admin Settings (gear icon on the bottom left).
2.- Go to 'Agents' under the 'User Management' section.
3.- Search for the Agent (user) that you want to enable the API Key for, and click on it. A more detailed view should be loaded. (don't click edit, click on the users name)
4.- Go to the 'Permissions' tab.
5.- Enable the Api Key.

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-freshservice
baton-freshservice
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_DOMAIN=<domain> -e BATON_API_KEY=<apiKey> ghcr.io/conductorone/baton-freshservice:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-freshservice/cmd/baton-freshservice@main

baton-freshservice

baton resources
```

# Running locally

```
baton-freshservice --api-key Xswedcvfrtgbyhnmju --domain conductorone
```

# Data Model

`baton-freshservice` will pull down information about the following resources:
- Users
- Groups
- Roles
- Requester Groups

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually
building spreadsheets. We welcome contributions, and ideas, no matter how
small&mdash;our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-freshservice` Command Line Usage

```
baton-freshservice

Usage:
  baton-freshservice [flags]
  baton-freshservice [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --api-key string         required: The api key for your account. ($BATON_API_KEY)
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --domain string          required: The domain for your account. ($BATON_DOMAIN)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-freshservice
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning           This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync         This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing              This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                version for baton-freshservice

Use "baton-freshservice [command] --help" for more information about a command.
```

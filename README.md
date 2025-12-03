# Meteor Addon Scanner

A tool to create a list of Meteor Client addons from github.

Check out the [Meteor Addon List](https://www.meteoraddons.com)

<a href="https://discord.gg/XU7Y9G46KD"><img src="https://invidget.switchblade.xyz/XU7Y9G46KD"></a>

## Usage

1. Create a **.env** file with a value **KEY** with a github API key with read access to public repositories
2. create a config.json file

```json
{
  "repo-blacklist": [],
  "developer-blacklist": [],
  "verified": [],
  "minimum_minecraft_version": null,
  "ignore_archived": false,
  "ignore_forks": false
}
```

3. Run the following command

```bash
scanner config.json addons.json
```

## Output

```json
[
  {
    "name": "string",
    "description": "string",
    "mc_version": "string",
    "authors": ["string"],
    "features": {
      "modules": ["string"],
      "commands": ["string"],
      "hud_elements": ["string"],
      "custom_screens": ["string"],
      "feature_count": 0
    },
    "verified": false,
    "repo": {
      "id": "string",
      "owner": "string",
      "name": "string",
      "archived": false,
      "fork": true,
      "stars": 0,
      "downloads": 0,
      "last_update": "string RFC3339",
      "creation_date": "string RFC3339"
    },
    "links": {
      "github": "string",
      "downloads": ["asset-1", "asset-2"],
      "discord": "string",
      "latest_release": "string",
      "homepage": "string",
      "icon": "string"
    },
    "custom": {
      "description": "string",
      "supported_versions": ["x.x.x", "x.x.x"],
      "icon": "string",
      "discord": "string",
      "homepage": "string"
    }
  }
]
```

## Custom Properties

The scanner automatically pulls info from GitHub, but it might not always be accurate or exactly how you want it. To fix or customize that data, you can manually add your own values.

To do that, create the file `meteor-addon-list.json` in the root directory of your addon, and add the fields you wish to override:

```json
{
  "description": "A short description of your addon.",
  "tags": [
    "PvP",
    "Utility",
    "Theme",
    "Render",
    "Movement",
    "Building",
    "World",
    "Misc",
    "QoL",
    "Exploit",
    "Fun",
    "Automation"
  ],
  "supported_versions": ["1.21.7", "1.21.8"],
  "icon": "https://example.com/icon.png",
  "discord": "https://discord.gg/yourserver",
  "homepage": "https://example.com"
}
```

_This tool is based on [AntiCope](https://github.com/AntiCope/anticope.ml)_

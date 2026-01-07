# Meteor Addon Scanner

A tool to create a list of Meteor Client addons from github.

Check out the [Meteor Addon List](https://meteoraddons.com)

<a href="https://discord.gg/XU7Y9G46KD"><img src="https://invidget.switchblade.xyz/XU7Y9G46KD"></a>

## Usage

1. Create a **.env** file with a value **KEY** with a github API key with read access to public repositories
2. create a config.json file

```json
{
  "repo-blacklist": [],
  "developer-blacklist": [],
  "verified_addons": {
    "verified": [],
    "minimum_mc_version": "1.20",
    "validate_forks": true
  },
  "module_descriptions": {
    "fetch": true,
    "only_verified": true,
    "minimum_star_count": 0
  },
  "require_mc_version": false,
  "ignore_archived": false,
  "ignore_forks": false,
  "suspicion_triggers": {
    "name_len": 50,
    "description_len": 333,
    "feature_count": 1000,
    "supported_versions": 15
  }
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
      "modules": [
        {
          "name": "name",
          "description": "description here"
        }
      ],
      "commands": [
        {
          "name": "name",
          "description": "description here"
        }
      ],
      "hud_elements": [
        {
          "name": "name",
          "description": "description here"
        }
      ],
      "tabs": ["string"],
      "themes": ["string"],
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
  "homepage": "https://example.com",
  "feature_directories": {
    "commands": ["modules/commands"],
    "modules": ["modules/general"],
    "hud_elements": ["modules/hud"]
  }
}
```

### Feature Directories

`feature_directories` tells the scanner where to find your Java files for modules, commands, and HUD elements.

- Start from the entrypoint package, remove the class name -> base path (`cqb13/NumbyHack`).
- List directories **relative to the base path**:
- Only list directories, not files. Use forward slashes `/` and no leading or ending slash.

_This tool is based on [AntiCope](https://github.com/AntiCope/anticope.ml)_

# Meteor Addon Scanner

A tool to create a list of Meteor Client addons from github.

Check out the [Meteor Addon List](https://meteor-addons.cqb13.dev/)

## Usage

1. Create a **.env** file with a value **KEY** with a github API key with read access to public repositories
2. Create a text file with full names of github repositories separated by new lines
3. Run the following command

```bash
scanner anticope-verified.txt addons.json
```

## Output

```json
[
  {
    "name": "string",
    "description": "string",
    "mc_version": "string",
    "authors": ["string"],
    "features": ["string", "string"],
    "feature_count": 0,
    "verified": false,
    "repo": {
      "id": "string",
      "owner": "string",
      "name": "string",
      "archived": false,
      "fork": true,
      "stars": 0,
      "downloads": 0,
      "last_update": "string",
      "creation_date": "string"
    },
    "links": {
      "github": "string",
      "download": "string",
      "discord": "string",
      "homepage": "string",
      "icon": "string"
    }
  }
]
```

_This tool is based on [AntiCope](https://github.com/AntiCope/anticope.ml)_

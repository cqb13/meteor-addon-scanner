import json

with open("temp.json", "r", encoding="utf-8") as file:
    data = json.load(file)

    print(len(data['items']))

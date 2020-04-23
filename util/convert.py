import json

ports = {}
with open('../input/tunnels.json') as json_file:
    data = json.load(json_file)
    for site in data:
        for dc in data[site]:
            for encoded in data[site][dc]:
                sections = encoded.split('_')
                print(sections[3] + ':' + sections[4], end=',')
                ports[sections[4]] = True

print("ports:")
for k in ports.keys():
    print(":" + k, end=',')

[
    {
        "portMappings": [
            {
                "containerPort": ${port}
            }
        ],
        "mountPoints": ${jsonencode([ for m in mounts: { "sourceVolume": m.src, "containerPath": m.dest, "readOnly": true }])},
        "environment": ${jsonencode([ for e in env: { "name": e.name, "value": e.value }])},
        "image": "${image}",
        "name": "${name}",
        "command": ${jsonencode(command)}
    }
]

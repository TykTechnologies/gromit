#!/usr/bin/env zsh

td=${1?"no task definition"}

old_td=$(aws ecs describe-task-definition --task-definition $td  \
      --query '{  containerDefinitions: taskDefinition.containerDefinitions,
                  family: taskDefinition.family,
                  executionRoleArn: taskDefinition.executionRoleArn,
                  networkMode: taskDefinition.networkMode,
                  volumes: taskDefinition.volumes,
                  placementConstraints: taskDefinition.placementConstraints,
                  requiresCompatibilities: taskDefinition.requiresCompatibilities,
                  cpu: taskDefinition.cpu,
                  memory: taskDefinition.memory}')
new_td=${old_td/tyk-analytics:federation-test/tyk-analytics:master}
print $new_td
aws ecs register-task-definition --family $td --cli-input-json "$new_td"
aws ecs update-service --cluster federation-test --service tyk-analytics --task-definition $td

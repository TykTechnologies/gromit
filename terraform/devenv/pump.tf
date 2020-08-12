# Pump

data "template_file" "pump" {
  template = templatefile("templates/cd-awsvpc.tpl",
    { port      = 3000,
      name      = local.pump_name,
      log_group = "internal",
      image     = local.tyk-pump_image,
      command   = ["--conf=/conf/tyk-pump.conf"],
      mounts = [
        { src = "config", dest = "/conf" }
      ],
      env = [],
  region = data.terraform_remote_state.base.outputs.region })
}

resource "aws_ecs_task_definition" "pump" {
  family                   = local.pump_name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  execution_role_arn       = data.aws_iam_role.ecs_task_execution_role.arn
  cpu                      = 256
  memory                   = 512

  container_definitions = data.template_file.pump.rendered

  volume {
    name = "config"

    efs_volume_configuration {
      file_system_id = data.terraform_remote_state.base.outputs.config_efs
      root_directory = "/${var.name}/tyk-pump"
    }
  }

  tags = local.common_tags
}

resource "aws_service_discovery_service" "pump" {
  name = "pump"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.internal.id

    dns_records {
      ttl  = 60
      type = "A"
    }
    routing_policy = "MULTIVALUE"
  }
}

resource "aws_ecs_service" "pump" {
  name            = local.pump_name
  cluster         = aws_ecs_cluster.env.id
  task_definition = aws_ecs_task_definition.pump.id
  desired_count   = 1
  launch_type     = "FARGATE"
  # Needed for EFS
  platform_version = "1.4.0"

  network_configuration {
    subnets          = data.aws_subnet_ids.public.ids
    security_groups  = [aws_security_group.pump.id]
    assign_public_ip = true
  }

  service_registries {
    registry_arn = aws_service_discovery_service.pump.arn
  }

  tags = local.common_tags
}

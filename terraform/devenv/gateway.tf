# Gateway

data "template_file" "gateway" {
  template = templatefile("templates/cd-awsvpc.tpl",
    { port      = 8080,
      name      = local.gw_name,
      log_group = "internal",
      image     = var.tyk,
      command   = ["--conf=/conf/tyk.conf"],
      mounts = [
        { src = "config", dest = "/conf" }
      ],
      env = [],
  region = var.region })
}

resource "aws_ecs_task_definition" "gateway" {
  family                   = local.gw_name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  execution_role_arn       = data.aws_iam_role.ecs_task_execution_role.arn
  cpu                      = 256
  memory                   = 512

  container_definitions = data.template_file.gateway.rendered

  volume {
    name = "config"

    efs_volume_configuration {
      file_system_id = var.config_efs
      root_directory = "/default/tyk"
    }
  }

  tags = local.common_tags
}

resource "aws_service_discovery_service" "gateway" {
  name = "gateway"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.internal.id

    dns_records {
      ttl  = 60
      type = "A"
    }
  }
}

resource "aws_ecs_service" "gateway" {
  name            = local.gw_name
  cluster         = aws_ecs_cluster.env.id
  task_definition = aws_ecs_task_definition.gateway.id
  desired_count   = 1
  launch_type     = "FARGATE"
  # Needed for EFS
  platform_version = "1.4.0"

  network_configuration {
    subnets          = data.aws_subnet_ids.public.ids
    security_groups  = [aws_security_group.gateway.id]
    assign_public_ip = true
  }

  service_registries {
    registry_arn = aws_service_discovery_service.gateway.arn
  }

  tags = local.common_tags
}

# Redis

resource "aws_security_group" "redis" {
  name        = "redis"
  description = "Allow traffic from anywhere in the vpc"
  vpc_id      = data.aws_vpc.devenv.id


  ingress {
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = [ data.aws_vpc.devenv.cidr_block ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

resource "aws_service_discovery_service" "redis" {
  name = "redis"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.internal.id

    dns_records {
      ttl  = 60
      type = "A"
    }
  }
}

resource "aws_ecs_service" "redis" {
  name            = local.redis_name
  cluster         = aws_ecs_cluster.env.id
  task_definition = aws_ecs_task_definition.redis.id
  desired_count   = 1
  launch_type     = "FARGATE"
  # Needed for EFS
  platform_version = "1.4.0"

  network_configuration {
    subnets          = data.aws_subnet_ids.private.ids
    security_groups  = [aws_security_group.redis.id]
  }

  service_registries {
    registry_arn = aws_service_discovery_service.redis.arn
    port = 6379
  }

  tags = local.common_tags
}

data "template_file" "redis" {
  template = templatefile("templates/cd-awsvpc.tpl", {
    port      = 6379,
    name      = "redis"
    mounts    = [],
    env       = [],
    command   = [],
    log_group = "internal",
    image     = "redis",
    region    = var.region,
  })
}

resource "aws_ecs_task_definition" "redis" {
  family                   = "redis"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  execution_role_arn       = data.aws_iam_role.ecs_task_execution_role.arn
  cpu                      = 256
  memory                   = 512

  container_definitions = data.template_file.redis.rendered

  tags = local.common_tags
}

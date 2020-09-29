# Dashboard

data "template_file" "tdb" {
  template = templatefile("templates/interactive.tpl",
    { port      = 22,
      name      = var.name,
      log_group = "pods",
      image     = var.image,
      command   = var.cmdline,
      mounts    = [],
      env       = [],
  region = data.terraform_remote_state.base.outputs.region })
}

resource "aws_ecs_task_definition" "tdb" {
  family                   = var.name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  execution_role_arn       = data.aws_iam_role.ecs_task_execution_role.arn
  cpu                      = 256
  memory                   = 512

  container_definitions = data.template_file.tdb.rendered
  tags                  = local.common_tags
}


resource "aws_security_group" "tdb" {
  name        = "tdb-${var.name}"
  description = "Ad-hoc for Tyk debugger"
  vpc_id      = data.terraform_remote_state.infra.outputs.vpc_id


  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

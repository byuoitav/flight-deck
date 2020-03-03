terraform {
  backend "s3" {
    bucket     = "terraform-state-storage-586877430255"
    lock_table = "terraform-state-lock-586877430255"
    region     = "us-west-2"

    // THIS MUST BE UNIQUE
    key = "flight-deck.tfstate"
  }
}

provider "aws" {
  region = "us-west-2"
}

data "aws_ssm_parameter" "eks_cluster_endpoint" {
  name = "/eks/av-cluster-endpoint"
}

provider "kubernetes" {
  host = data.aws_ssm_parameter.eks_cluster_endpoint.value
}

// pull all env vars out of ssm
data "aws_ssm_parameter" "deployment_key" {
  name = "/flight-deck/all/deployment-key"
}

data "aws_ssm_parameter" "dev_couch_address" {
  name = "/env/dev-couch-address"
}

data "aws_ssm_parameter" "dev_couch_username" {
  name = "/env/dev-couch-username"
}

data "aws_ssm_parameter" "dev_couch_password" {
  name = "/env/dev-couch-password"
}

data "aws_ssm_parameter" "prd_couch_address" {
  name = "/env/couch-address"
}

data "aws_ssm_parameter" "prd_couch_username" {
  name = "/env/couch-username"
}

data "aws_ssm_parameter" "prd_couch_password" {
  name = "/env/couch-password"
}

data "aws_ssm_parameter" "docker_github_password" {
  name = "/flight-deck/all/docker-github-password"
}

data "aws_ssm_parameter" "docker_github_username" {
  name = "/flight-deck/all/docker-github-username"
}

data "aws_ssm_parameter" "elk_event_api" {
  name = "/flight-deck/all/elk-event-api"
}

data "aws_ssm_parameter" "ldap_password" {
  name = "/flight-deck/all/ldap-password"
}

data "aws_ssm_parameter" "ldap_username" {
  name = "/flight-deck/all/ldap-username"
}

data "aws_ssm_parameter" "pi_username" {
  name = "/flight-deck/all/pi-username"
}

resource "aws_s3_bucket" "bucket" {
  bucket = "av-flight-deck"

  tags = {
    repo_url         = "https://github.com/byuoitav/flight-deck"
    team             = "AV Engineering"
    data-sensitivity = "confidential"
  }
}

data "aws_iam_policy_document" "policy" {
  statement {
    actions = [
      "s3:ListAllMyBuckets",
      "s3:GetBucketLocation",
    ]
    resources = [
      "arn:aws:s3:::*"
    ]
  }

  statement {
    actions = [
      "s3:ListBucket",
    ]
    resources = [
      "arn:aws:s3:::${aws_s3_bucket.bucket.id}"
    ]
  }

  statement {
    actions = [
      "s3:*",
    ]

    resources = [
      "arn:aws:s3:::${aws_s3_bucket.bucket.id}",
      "arn:aws:s3:::${aws_s3_bucket.bucket.id}/*",
    ]
  }
}

module "stg_deployment" {
  source = "github.com/byuoitav/terraform//modules/kubernetes-deployment"

  // required
  name           = "flight-deck-stg"
  image          = "byuoitav/flight-deck"
  image_version  = "development"
  container_port = 8008
  repo_url       = "https://github.com/byuoitav/flight-deck"

  // optional
  iam_policy_doc = data.aws_iam_policy_document.policy.json
  public_urls    = ["flight-deck-stg.av.byu.edu"]
  container_env = {
    "AWS_BUCKET_REGION"          = aws_s3_bucket.bucket.region
    "AWS_DEPLOYMENT_KEY"         = data.aws_ssm_parameter.deployment_key.value
    "DB_ADDRESS"                 = data.aws_ssm_parameter.dev_couch_address.value
    "DB_USERNAME"                = data.aws_ssm_parameter.dev_couch_username.value
    "DB_PASSWORD"                = data.aws_ssm_parameter.dev_couch_password.value
    "DOCKER_GITHUB_PASSWORD"     = data.aws_ssm_parameter.docker_github_password.value
    "DOCKER_GITHUB_USERNAME"     = data.aws_ssm_parameter.docker_github_username.value
    "ELASTIC_API_EVENTS"         = data.aws_ssm_parameter.elk_event_api.value
    "LDAP_PASSWORD"              = data.aws_ssm_parameter.ldap_password.value
    "LDAP_USERNAME"              = data.aws_ssm_parameter.ldap_username.value
    "PI_SSH_USERNAME"            = data.aws_ssm_parameter.pi_username.value
    "RASPI_DEPLOYMENT_S3_BUCKET" = aws_s3_bucket.bucket.id
    "STOP_REPLICATION"           = "true"
  }
}

module "prd_deployment" {
  source = "github.com/byuoitav/terraform//modules/kubernetes-deployment"

  // required
  name           = "flight-deck-prd"
  image          = "byuoitav/flight-deck"
  image_version  = "development"
  container_port = 8008
  repo_url       = "https://github.com/byuoitav/flight-deck"

  // optional
  iam_policy_doc = data.aws_iam_policy_document.policy.json
  public_urls    = ["flight-deck-prd.av.byu.edu"]
  container_env = {
    "AWS_BUCKET_REGION"          = aws_s3_bucket.bucket.region
    "AWS_DEPLOYMENT_KEY"         = data.aws_ssm_parameter.deployment_key.value
    "DB_ADDRESS"                 = data.aws_ssm_parameter.prd_couch_address.value
    "DB_USERNAME"                = data.aws_ssm_parameter.prd_couch_username.value
    "DB_PASSWORD"                = data.aws_ssm_parameter.prd_couch_password.value
    "DOCKER_GITHUB_PASSWORD"     = data.aws_ssm_parameter.docker_github_password.value
    "DOCKER_GITHUB_USERNAME"     = data.aws_ssm_parameter.docker_github_username.value
    "ELASTIC_API_EVENTS"         = data.aws_ssm_parameter.elk_event_api.value
    "LDAP_PASSWORD"              = data.aws_ssm_parameter.ldap_password.value
    "LDAP_USERNAME"              = data.aws_ssm_parameter.ldap_username.value
    "PI_SSH_USERNAME"            = data.aws_ssm_parameter.pi_username.value
    "RASPI_DEPLOYMENT_S3_BUCKET" = aws_s3_bucket.bucket.id
    "STOP_REPLICATION"           = "true"
  }
}

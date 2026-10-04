data "aws_caller_identity" "current" {}

resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = [
    "sts.amazonaws.com"
  ]

  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1"
  ]
}

# ECR pipeline role

resource "aws_iam_role" "github_ecr" {
  name = "bravo-github-ecr"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"

    Statement = [
      {
        Effect = "Allow"

        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }

        Action = "sts:AssumeRoleWithWebIdentity"

        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
            "token.actions.githubusercontent.com:sub" = "repo:bravo-eks@330665479/bravo-eks@1373795192:environment:production"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "github_ecr" {
  name = "bravo-github-ecr-policy"
  role = aws_iam_role.github_ecr.id

  policy = jsonencode({
    Version = "2012-10-17"

    Statement = [
      {
        Sid    = "ECRAuth"
        Effect = "Allow"

        Action = [
          "ecr:GetAuthorizationToken"
        ]

        Resource = "*"
      },
      {
        Sid    = "ECRPushPull"
        Effect = "Allow"

        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload"
        ]

        Resource = values(module.ecr.repository_arns)
      }
    ]
  })
}

# Terraform PR pipeline role

resource "aws_iam_role" "github_terraform_pr" {
  name = "bravo-github-terraform-pr"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"

    Statement = [
      {
        Effect = "Allow"

        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }

        Action = "sts:AssumeRoleWithWebIdentity"

        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
            "token.actions.githubusercontent.com:sub" = "repo:bravo-eks@330665479/bravo-eks@1373795192:pull_request"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "github_terraform_pr" {
  name = "bravo-github-terraform-pr-policy"
  role = aws_iam_role.github_terraform_pr.id

  policy = jsonencode({
    Version = "2012-10-17"

    Statement = [
      {
        Sid    = "TerraformStateRead"
        Effect = "Allow"

        Action = [
          "s3:ListBucket",
          "s3:GetObject"
        ]

        Resource = [
          aws_s3_bucket.terraform_state.arn,
          "${aws_s3_bucket.terraform_state.arn}/*"
        ]
      },
      {
        Sid    = "EC2Read"
        Effect = "Allow"

        Action = [
          "ec2:Describe*"
        ]

        Resource = "*"
      },
      {
        Sid    = "RDSRead"
        Effect = "Allow"

        Action = [
          "rds:Describe*",
          "rds:ListTagsForResource"
        ]

        Resource = "*"
      },
      {
        Sid    = "EKSRead"
        Effect = "Allow"

        Action = [
          "eks:Describe*",
          "eks:List*"
        ]

        Resource = "*"
      },
      {
        Sid    = "SQSRead"
        Effect = "Allow"

        Action = [
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:ListQueues",
          "sqs:ListQueueTags"
        ]

        Resource = "*"
      },
      {
        Sid    = "IAMRead"
        Effect = "Allow"

        Action = [
          "iam:Get*",
          "iam:List*"
        ]

        Resource = "*"
      },
      {
        Sid    = "SecretsManagerRead"
        Effect = "Allow"

        Action = [
          "secretsmanager:DescribeSecret",
          "secretsmanager:GetResourcePolicy",
          "secretsmanager:GetSecretValue",
          "secretsmanager:ListSecretVersionIds"
        ]

        Resource = "*"
      },
      {
        Sid    = "KMSRead"
        Effect = "Allow"

        Action = [
          "kms:DescribeKey",
          "kms:GetKeyPolicy",
          "kms:GetKeyRotationStatus",
          "kms:ListAliases",
          "kms:ListResourceTags"
        ]

        Resource = "*"
      }
    ]
  })
}

# Terraform deploy pipeline role

resource "aws_iam_role" "github_terraform_deploy" {
  name = "bravo-github-terraform-deploy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"

    Statement = [
      {
        Effect = "Allow"

        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }

        Action = "sts:AssumeRoleWithWebIdentity"

        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
            "token.actions.githubusercontent.com:sub" = "repo:bravo-eks@330665479/bravo-eks@1373795192:ref:refs/heads/main"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "github_terraform_deploy" {
  name = "bravo-github-terraform-deploy-policy"
  role = aws_iam_role.github_terraform_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"

    Statement = [
      {
        Sid    = "TerraformState"
        Effect = "Allow"

        Action = [
          "s3:ListBucket",
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]

        Resource = [
          aws_s3_bucket.terraform_state.arn,
          "${aws_s3_bucket.terraform_state.arn}/*"
        ]
      },

      # VPC

      {
        Sid    = "VPCRead"
        Effect = "Allow"

        Action = [
          "ec2:Describe*"
        ]

        Resource = "*"
      },
      {
        Sid    = "VPCManage"
        Effect = "Allow"

        Action = [
          "ec2:CreateVpc",
          "ec2:DeleteVpc",
          "ec2:ModifyVpcAttribute",
          "ec2:CreateSubnet",
          "ec2:DeleteSubnet",
          "ec2:ModifySubnetAttribute",
          "ec2:CreateInternetGateway",
          "ec2:DeleteInternetGateway",
          "ec2:AttachInternetGateway",
          "ec2:DetachInternetGateway",
          "ec2:CreateRouteTable",
          "ec2:DeleteRouteTable",
          "ec2:AssociateRouteTable",
          "ec2:DisassociateRouteTable",
          "ec2:CreateRoute",
          "ec2:ReplaceRoute",
          "ec2:DeleteRoute",
          "ec2:AllocateAddress",
          "ec2:ReleaseAddress",
          "ec2:CreateNatGateway",
          "ec2:DeleteNatGateway",
          "ec2:CreateSecurityGroup",
          "ec2:DeleteSecurityGroup",
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:RevokeSecurityGroupIngress",
          "ec2:AuthorizeSecurityGroupEgress",
          "ec2:RevokeSecurityGroupEgress",
          "ec2:CreateTags",
          "ec2:DeleteTags"
        ]

        Resource = "*"
      },

      # RDS

      {
        Sid    = "RDSRead"
        Effect = "Allow"

        Action = [
          "rds:Describe*",
          "rds:ListTagsForResource"
        ]

        Resource = "*"
      },
      {
        Sid    = "RDSManage"
        Effect = "Allow"

        Action = [
          "rds:CreateDBInstance",
          "rds:ModifyDBInstance",
          "rds:DeleteDBInstance",
          "rds:CreateDBSubnetGroup",
          "rds:ModifyDBSubnetGroup",
          "rds:DeleteDBSubnetGroup",
          "rds:AddTagsToResource",
          "rds:RemoveTagsFromResource"
        ]

        Resource = "*"
      },

      # EKS

      {
        Sid    = "EKSRead"
        Effect = "Allow"

        Action = [
          "eks:Describe*",
          "eks:List*"
        ]

        Resource = "*"
      },
      {
        Sid    = "EKSManage"
        Effect = "Allow"

        Action = [
          "eks:CreateCluster",
          "eks:UpdateClusterConfig",
          "eks:UpdateClusterVersion",
          "eks:DeleteCluster",
          "eks:CreateNodegroup",
          "eks:UpdateNodegroupConfig",
          "eks:UpdateNodegroupVersion",
          "eks:DeleteNodegroup",
          "eks:CreateAddon",
          "eks:UpdateAddon",
          "eks:DeleteAddon",
          "eks:TagResource",
          "eks:UntagResource"
        ]

        Resource = "*"
      },

      # SQS

      {
        Sid    = "SQSRead"
        Effect = "Allow"

        Action = [
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:ListQueues",
          "sqs:ListQueueTags"
        ]

        Resource = "*"
      },
      {
        Sid    = "SQSManage"
        Effect = "Allow"

        Action = [
          "sqs:CreateQueue",
          "sqs:DeleteQueue",
          "sqs:SetQueueAttributes",
          "sqs:TagQueue",
          "sqs:UntagQueue"
        ]

        Resource = "*"
      },

      # IAM

      {
        Sid    = "IAMRead"
        Effect = "Allow"

        Action = [
          "iam:Get*",
          "iam:List*"
        ]

        Resource = "*"
      },
      {
        Sid    = "IAMManage"
        Effect = "Allow"

        Action = [
          "iam:CreateRole",
          "iam:DeleteRole",
          "iam:UpdateAssumeRolePolicy",
          "iam:TagRole",
          "iam:UntagRole",
          "iam:PutRolePolicy",
          "iam:DeleteRolePolicy",
          "iam:CreatePolicy",
          "iam:DeletePolicy",
          "iam:CreatePolicyVersion",
          "iam:DeletePolicyVersion",
          "iam:SetDefaultPolicyVersion",
          "iam:AttachRolePolicy",
          "iam:DetachRolePolicy",
          "iam:PassRole",
          "iam:TagPolicy",
          "iam:UntagPolicy",
          "iam:CreateServiceLinkedRole"
        ]

        Resource = "*"
      },

      # ECR

      {
        Sid    = "ECRManage"
        Effect = "Allow"

        Action = [
          "ecr:CreateRepository",
          "ecr:DeleteRepository",
          "ecr:DescribeRepositories",
          "ecr:ListTagsForResource",
          "ecr:TagResource",
          "ecr:UntagResource",
          "ecr:PutLifecyclePolicy",
          "ecr:GetLifecyclePolicy",
          "ecr:DeleteLifecyclePolicy"
        ]

        Resource = "*"
      },

      # KMS

      {
        Sid    = "KMSManage"
        Effect = "Allow"

        Action = [
          "kms:CreateKey",
          "kms:DescribeKey",
          "kms:EnableKeyRotation",
          "kms:GetKeyRotationStatus",
          "kms:CreateAlias",
          "kms:DeleteAlias",
          "kms:UpdateAlias",
          "kms:ListAliases",
          "kms:TagResource",
          "kms:UntagResource",
          "kms:ListResourceTags",
          "kms:ScheduleKeyDeletion",
          "kms:GetKeyPolicy",
          "kms:CreateGrant",
          "kms:CancelKeyDeletion"
        ]

        Resource = "*"
      },

      # Secrets Manager

      {
        Sid    = "SecretsManagerManage"
        Effect = "Allow"

        Action = [
          "secretsmanager:CreateSecret",
          "secretsmanager:TagResource"
        ]

        Resource = "*"
      },

      # EKS OIDC

      {
        Sid    = "OIDCManage"
        Effect = "Allow"

        Action = [
          "iam:CreateOpenIDConnectProvider",
          "iam:GetOpenIDConnectProvider",
          "iam:UpdateOpenIDConnectProviderThumbprint",
          "iam:AddClientIDToOpenIDConnectProvider",
          "iam:RemoveClientIDFromOpenIDConnectProvider",
          "iam:DeleteOpenIDConnectProvider",
          "iam:TagOpenIDConnectProvider",
          "iam:UntagOpenIDConnectProvider"
        ]

        Resource = "*"
      },
    ]
  })
}
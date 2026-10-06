# Order Service - IRSA Trust Policy

data "aws_iam_policy_document" "order_service_assume_role" {
  statement {
    sid    = "AllowOrderServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.kubernetes_namespace}:${var.order_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "order_service" {
  name               = "${var.name_prefix}-order-service"
  description        = "IAM role assumed by the order-service Kubernetes workload using IRSA."
  assume_role_policy = data.aws_iam_policy_document.order_service_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-order-service"
      Component = "order-service"
    }
  )
}


# Payment Service - IRSA Trust Policy

data "aws_iam_policy_document" "payment_service_assume_role" {
  statement {
    sid    = "AllowPaymentServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.kubernetes_namespace}:${var.payment_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "payment_service" {
  name               = "${var.name_prefix}-payment-service"
  description        = "IAM role assumed by the payment-service Kubernetes workload using IRSA."
  assume_role_policy = data.aws_iam_policy_document.payment_service_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-payment-service"
      Component = "payment-service"
    }
  )
}


# Shipping Service - IRSA Trust Policy

data "aws_iam_policy_document" "shipping_service_assume_role" {
  statement {
    sid    = "AllowShippingServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.kubernetes_namespace}:${var.shipping_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "shipping_service" {
  name               = "${var.name_prefix}-shipping-service"
  description        = "IAM role assumed by the shipping-service Kubernetes workload using IRSA."
  assume_role_policy = data.aws_iam_policy_document.shipping_service_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-shipping-service"
      Component = "shipping-service"
    }
  )
}


# Worker - IRSA Trust Policy

data "aws_iam_policy_document" "worker_assume_role" {
  statement {
    sid    = "AllowWorkerServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.kubernetes_namespace}:${var.worker_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "worker" {
  name               = "${var.name_prefix}-worker"
  description        = "IAM role assumed by the worker Kubernetes workload using IRSA."
  assume_role_policy = data.aws_iam_policy_document.worker_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-worker"
      Component = "worker"
    }
  )
}


# ExternalDNS - IRSA Trust Policy

data "aws_iam_policy_document" "external_dns_assume_role" {
  statement {
    sid    = "AllowExternalDNSServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.external_dns_namespace}:${var.external_dns_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "external_dns" {
  name               = "${var.name_prefix}-external-dns"
  description        = "IAM role assumed by ExternalDNS using IRSA."
  assume_role_policy = data.aws_iam_policy_document.external_dns_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-external-dns"
      Component = "external-dns"
    }
  )
}


# ExternalDNS - Route53 Policy

data "aws_iam_policy_document" "external_dns_route53" {

  statement {
    sid    = "ManageBravoOrdersDNSRecords"
    effect = "Allow"

    actions = [
      "route53:ChangeResourceRecordSets",
      "route53:ListResourceRecordSets",
      "route53:ListTagsForResources"
    ]

    resources = [
      "arn:aws:route53:::hostedzone/Z094258811V8WSH58ZFW5"
    ]
  }

  statement {
    sid    = "ListHostedZones"
    effect = "Allow"

    actions = [
      "route53:ListHostedZones"
    ]

    resources = [
      "*"
    ]
  }
}

resource "aws_iam_policy" "external_dns_route53" {
  name        = "${var.name_prefix}-external-dns-route53"
  description = "Allows ExternalDNS to manage Route53 records for bravoorders.com."
  policy      = data.aws_iam_policy_document.external_dns_route53.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-external-dns-route53"
      Component = "external-dns"
    }
  )
}

resource "aws_iam_role_policy_attachment" "external_dns_route53" {
  role       = aws_iam_role.external_dns.name
  policy_arn = aws_iam_policy.external_dns_route53.arn
}


# SQS Producer Policy

data "aws_iam_policy_document" "sqs_producer" {
  statement {
    sid    = "SendMessages"
    effect = "Allow"

    actions = [
      "sqs:SendMessage"
    ]

    resources = var.sqs_queue_arns
  }
}

resource "aws_iam_policy" "sqs_producer" {
  name        = "${var.name_prefix}-sqs-producer"
  description = "Least-privilege SQS SendMessage permission for event-producing workloads."
  policy      = data.aws_iam_policy_document.sqs_producer.json

  tags = merge(
    var.tags,
    {
      Name = "${var.name_prefix}-sqs-producer"
    }
  )
}


# Attach producer policy to each producer role

resource "aws_iam_role_policy_attachment" "order_service_sqs" {
  role       = aws_iam_role.order_service.name
  policy_arn = aws_iam_policy.sqs_producer.arn
}

resource "aws_iam_role_policy_attachment" "payment_service_sqs" {
  role       = aws_iam_role.payment_service.name
  policy_arn = aws_iam_policy.sqs_producer.arn
}

resource "aws_iam_role_policy_attachment" "shipping_service_sqs" {
  role       = aws_iam_role.shipping_service.name
  policy_arn = aws_iam_policy.sqs_producer.arn
}


# Worker SQS Consumer Policy

data "aws_iam_policy_document" "worker_sqs" {
  statement {
    sid    = "ConsumeMessages"
    effect = "Allow"

    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:ChangeMessageVisibility",
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl"
    ]

    resources = var.sqs_queue_arns
  }
}

resource "aws_iam_policy" "worker_sqs" {
  name        = "${var.name_prefix}-worker-sqs"
  description = "Least-privilege SQS consumer permissions for the worker workload."
  policy      = data.aws_iam_policy_document.worker_sqs.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-worker-sqs"
      Component = "worker"
    }
  )
}

resource "aws_iam_role_policy_attachment" "worker_sqs" {
  role       = aws_iam_role.worker.name
  policy_arn = aws_iam_policy.worker_sqs.arn
}

# External Secrets Operator - IRSA Trust Policy

data "aws_iam_policy_document" "external_secrets_assume_role" {
  statement {
    sid    = "AllowExternalSecretsServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.external_secrets_namespace}:${var.external_secrets_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "external_secrets" {
  name               = "${var.name_prefix}-external-secrets"
  description        = "IAM role assumed by External Secrets Operator using IRSA."
  assume_role_policy = data.aws_iam_policy_document.external_secrets_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-external-secrets"
      Component = "external-secrets"
    }
  )
}

# External Secrets Operator - Secrets Manager Policy

data "aws_iam_policy_document" "external_secrets" {
  statement {
    sid    = "ReadSecrets"
    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
      "secretsmanager:DescribeSecret"
    ]

    resources = var.external_secrets_secret_arns
  }
}

resource "aws_iam_policy" "external_secrets" {
  name        = "${var.name_prefix}-external-secrets"
  description = "Allows External Secrets Operator to read approved AWS Secrets Manager secrets."
  policy      = data.aws_iam_policy_document.external_secrets.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-external-secrets"
      Component = "external-secrets"
    }
  )
}

resource "aws_iam_role_policy_attachment" "external_secrets" {
  role       = aws_iam_role.external_secrets.name
  policy_arn = aws_iam_policy.external_secrets.arn
}

# Velero - IRSA Trust Policy

data "aws_iam_policy_document" "velero_assume_role" {
  statement {
    sid    = "AllowVeleroServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.velero_namespace}:${var.velero_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "velero" {
  name               = "${var.name_prefix}-velero"
  description        = "IAM role assumed by Velero using IRSA for Kubernetes backups and EBS snapshots."
  assume_role_policy = data.aws_iam_policy_document.velero_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-velero"
      Component = "velero"
    }
  )
}

# Velero - AWS Permissions

data "aws_iam_policy_document" "velero" {
  statement {
    sid    = "EBSVolumeSnapshots"
    effect = "Allow"

    actions = [
      "ec2:DescribeVolumes",
      "ec2:DescribeSnapshots",
      "ec2:CreateSnapshot",
      "ec2:DeleteSnapshot",
      "ec2:CreateTags"
    ]

    resources = ["*"]
  }

  statement {
    sid    = "VeleroS3Bucket"
    effect = "Allow"

    actions = [
      "s3:GetObject",
      "s3:DeleteObject",
      "s3:PutObject",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts"
    ]

    resources = [
      "${var.velero_backup_bucket_arn}/*"
    ]
  }

  statement {
    sid    = "VeleroS3BucketList"
    effect = "Allow"

    actions = [
      "s3:ListBucket",
      "s3:GetBucketLocation"
    ]

    resources = [
      var.velero_backup_bucket_arn
    ]
  }
}

resource "aws_iam_role_policy" "velero" {
  name   = "${var.name_prefix}-velero"
  role   = aws_iam_role.velero.id
  policy = data.aws_iam_policy_document.velero.json
}

# KEDA - IRSA Trust Policy

data "aws_iam_policy_document" "keda_assume_role" {
  statement {
    sid    = "AllowKedaOperatorServiceAccount"
    effect = "Allow"

    actions = [
      "sts:AssumeRoleWithWebIdentity"
    ]

    principals {
      type = "Federated"

      identifiers = [
        var.eks_oidc_provider_arn
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:aud"

      values = [
        "sts.amazonaws.com"
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${var.eks_oidc_provider_url}:sub"

      values = [
        "system:serviceaccount:${var.keda_namespace}:${var.keda_service_account_name}"
      ]
    }
  }
}

resource "aws_iam_role" "keda" {
  name               = "${var.name_prefix}-keda"
  description        = "IAM role assumed by the KEDA operator using IRSA."
  assume_role_policy = data.aws_iam_policy_document.keda_assume_role.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-keda"
      Component = "keda"
    }
  )
}

data "aws_iam_policy_document" "keda_sqs" {
  statement {
    sid    = "ReadQueueDepth"
    effect = "Allow"

    actions = [
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl"
    ]

    resources = var.sqs_queue_arns
  }
}

resource "aws_iam_policy" "keda_sqs" {
  name        = "${var.name_prefix}-keda-sqs"
  description = "Least-privilege SQS queue inspection permissions for KEDA autoscaling."
  policy      = data.aws_iam_policy_document.keda_sqs.json

  tags = merge(
    var.tags,
    {
      Name      = "${var.name_prefix}-keda-sqs"
      Component = "keda"
    }
  )
}

resource "aws_iam_role_policy_attachment" "keda_sqs" {
  role       = aws_iam_role.keda.name
  policy_arn = aws_iam_policy.keda_sqs.arn
}

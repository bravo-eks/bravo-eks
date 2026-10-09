# EC2 Spot service-linked role required for Karpenter Spot provisioning.

resource "aws_iam_service_linked_role" "ec2_spot" {
  aws_service_name = "spot.amazonaws.com"
}
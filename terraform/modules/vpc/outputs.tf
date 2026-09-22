output "vpc_id" {
    description = "ID of the VPC"
    value = aws_vpc.bravo-vpc.id
}

output "vpc_cidr_block" {
    description = "CIDR block of the VPC"
    value = aws_vpc.bravo-vpc.cidr_block
}

output "public_subnet_ids" {
    description = "IDs of public subnets"
    value = aws_subnet.public[*].id
}

output "private_subnet_ids" {
    description = "IDs of private subnets"
    value = aws_subnet.private[*].id
}

output "nat_gateway_id" {
  description = "ID of the NAT Gateway"
  value       = var.enable_nat_gateway ? aws_nat_gateway.this[0].id : null
}
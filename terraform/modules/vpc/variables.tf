variable "name" {
  description = "Name prefix for VPC resources"
  type        = string
}

variable "cidr_blocks" {
    description = "CIDR block for the VPC"
    type = string
    default = "10.0.0.0/16"
}

variable "public_subnet_cidrs" {
    description = "CIDR blocks for public subnets"
    type = list(string)
    default = [ "10.0.1.0/24", "10.0.2.0/24" ]
}

variable "private_subnet_cidrs" {
    description = "CIDR blocks for private subnets"
    type = list(string)
    default = [ "10.0.3.0/24", "10.0.4.0/24" ]
}

variable "availability_zones" {
    description = "Availability Zones to use"
    type = list(string)
    default = [ "eu-west-1a", "eu-west-1b" ]
}

variable "enable_nat_gateway" {
    description = "Whether to create NAT gateway"
    type = bool
    default = true
}
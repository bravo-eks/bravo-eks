resource "aws_vpc" "bravo-vpc" {
    cidr_block = var.cidr_blocks
    enable_dns_hostnames = true
    enable_dns_support = true

    tags = {
            Name = var.name
    }
}

resource "aws_subnet" "public" {
    count = length(var.public_subnet_cidrs)

    vpc_id = aws_vpc.bravo-vpc.id
    cidr_block = var.public_subnet_cidrs[count.index]
    availability_zone = var.availability_zones[count.index]
    map_public_ip_on_launch = true

  tags = {
    Name = "${var.name}-public-${var.availability_zones[count.index]}"
  }
}

resource "aws_subnet" "private" {
    count = length(var.private_subnet_cidrs)

    vpc_id = aws_vpc.bravo-vpc.id
    cidr_block = var.private_subnet_cidrs[count.index]
    availability_zone = var.availability_zones[count.index]
    map_public_ip_on_launch = true

 tags = {
    Name = "${var.name}-private-${var.availability_zones[count.index]}"
  }
}

resource "aws_internet_gateway" "this" {
    vpc_id = aws_vpc.bravo-vpc.id

    tags = {
        Name = "${var.name}-igw"
    }
}

resource "aws_eip" "nat" {
    count = var.enable_nat_gateway ? 1 : 0

    domain = "vpc"

    tags = {
        Name = "${var.name}-nat-eip"
    }
}

resource "aws_nat_gateway" "this" {
    count = var.enable_nat_gateway ? 1 : 0

    allocation_id = aws_eip.nat[0].id
    subnet_id = aws_subnet.public[0].id

    depends_on = [ aws_internet_gateway.this ]

    tags = {
            Name = "${var.name}-nat"
    }
}

resource "aws_route_tabel" "public" {
    count = length(var.public_subnet_cidrs)

    vpc_id = aws_vpc.bravo.id

    route {
        cidr_block = "0.0.0.0/0"
        gateway_id = aws_internet_gateway.this.id
    }

  tags = {
    Name = "${var.name}-public-rt-${var.availability_zones[count.index]}"
  }
}

resource "aws_route_table_association" "public" {
    count = length(var.public_subnet_cidrs)

    subnet_id = aws_subnet.public[0].id
    route_table_id = aws_route_tabel.public[0].id

}

resource "aws_route_table" "private" {
  count = length(var.availability_zones)

  vpc_id = aws_vpc.this.id

  dynamic "route" {
    for_each = var.enable_nat_gateway ? [1] : []

    content {
      cidr_block     = "0.0.0.0/0"
      nat_gateway_id = aws_nat_gateway.this[0].id
    }
  }

  tags = {
    Name = "${var.name}-private-rt-${var.availability_zones[count.index]}"
  }
}

resource "aws_route_table_association" "private" {
    count = length(var.private_subnet_cidrs)

    subnet_id = aws_subnet.private[0].id
    route_table_id = aws_route_table.private.id
}






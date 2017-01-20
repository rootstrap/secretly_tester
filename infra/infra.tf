variable "aws_access_key" {}
variable "aws_secret_key" {}
variable "aws_region" {}
variable "instance_type" {}
variable "security_group_id" {}
variable "subnet_id" {}
variable "ami_id" {}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  region = "${var.aws_region}"
}

resource "aws_instance" "loadtest" {
  count = 3
  instance_type = "${var.instance_type}"
  ami = "${var.ami_id}"
  vpc_security_group_ids = ["${var.security_group_id}"]
  subnet_id = "${var.subnet_id}"
}

output "nodeip" {
  value = ["${aws_instance.loadtest.*.public_ip}"]
}
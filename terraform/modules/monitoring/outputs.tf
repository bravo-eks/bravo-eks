output "sns_topic_arn" {
  description = "ARN of the SNS topic used for CloudWatch alerts."
  value       = aws_sns_topic.alerts.arn
}

output "sns_topic_name" {
  description = "Name of the SNS topic used for CloudWatch alerts."
  value       = aws_sns_topic.alerts.name
}

output "alarm_names" {
  description = "Names of the configured CloudWatch alarms."
  value = [
    aws_cloudwatch_metric_alarm.rds_cpu.alarm_name,
    aws_cloudwatch_metric_alarm.rds_free_storage.alarm_name,
    aws_cloudwatch_metric_alarm.sqs_queue_depth.alarm_name,
    aws_cloudwatch_metric_alarm.sqs_dlq_messages.alarm_name
  ]
}
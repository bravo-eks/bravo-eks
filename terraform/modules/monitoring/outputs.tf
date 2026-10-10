output "alarm_names" {
  description = "Names of the configured CloudWatch alarms."

  value = [
    aws_cloudwatch_metric_alarm.rds_cpu.alarm_name,
    aws_cloudwatch_metric_alarm.rds_free_storage.alarm_name,
    aws_cloudwatch_metric_alarm.sqs_queue_depth.alarm_name,
    aws_cloudwatch_metric_alarm.sqs_dlq_messages.alarm_name
  ]
}

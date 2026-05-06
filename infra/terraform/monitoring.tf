# Phase 9 — operational alarms and optional cost notifications.
# SNS email subscriptions require the recipient to confirm via AWS's confirmation link.

resource "aws_sns_topic" "ops_alerts" {
  count = var.alarm_notification_email != "" ? 1 : 0
  name  = "${local.name_prefix}-ops-alerts"
}

resource "aws_sns_topic_subscription" "ops_email" {
  count     = var.alarm_notification_email != "" ? 1 : 0
  topic_arn = aws_sns_topic.ops_alerts[0].arn
  protocol  = "email"
  endpoint  = var.alarm_notification_email
}

resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  alarm_name          = "${local.name_prefix}-api-lambda-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = 300
  statistic           = "Sum"
  threshold           = 0
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = aws_lambda_function.api.function_name
  }

  alarm_description = "Booking API Lambda reported errors in a 5-minute window"
  alarm_actions     = var.alarm_notification_email != "" ? [aws_sns_topic.ops_alerts[0].arn] : []
}

resource "aws_cloudwatch_metric_alarm" "lambda_duration" {
  alarm_name          = "${local.name_prefix}-api-lambda-duration-avg"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "Duration"
  namespace           = "AWS/Lambda"
  period              = 300
  statistic           = "Average"
  threshold           = 5000
  treat_missing_data  = "notBreaching"

  dimensions = {
    FunctionName = aws_lambda_function.api.function_name
  }

  alarm_description = "Average API Lambda duration exceeded 5 seconds over 10 minutes (adjust for your SLO)"
  alarm_actions     = var.alarm_notification_email != "" ? [aws_sns_topic.ops_alerts[0].arn] : []
}

resource "aws_budgets_budget" "monthly" {
  count = var.cost_alert_email != "" ? 1 : 0

  name              = "${local.name_prefix}-monthly-spend"
  budget_type       = "COST"
  limit_amount      = var.monthly_budget_usd
  limit_unit        = "USD"
  time_period_start = var.budget_start_date
  time_period_end   = "2087-12-31_23:59"
  time_unit         = "MONTHLY"

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 80
    threshold_type             = "PERCENTAGE"
    notification_type          = "ACTUAL"
    subscriber_email_addresses = [var.cost_alert_email]
  }
}

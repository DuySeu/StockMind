import os
import boto3

from logsim import CustomLogger

# Database pool instance
log = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


def get_s3_client():
    return boto3.client(
        service_name="s3",
        region_name="ap-southeast-1",
    )

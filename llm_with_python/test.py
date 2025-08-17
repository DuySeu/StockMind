import boto3

# Step 1: Create an STS client with base credentials
sts_client = boto3.client("sts")

# Step 2: Assume the role
role_arn = "arn:aws:iam::130506138320:role/bedrock-cross-account-role"  # Replace with your role ARN
role_session_name = "BedrockSession"  # A name for the session

response = sts_client.assume_role(RoleArn=role_arn, RoleSessionName=role_session_name)

# Step 3: Extract temporary credentials
credentials = response["Credentials"]
access_key = credentials["AccessKeyId"]
secret_key = credentials["SecretAccessKey"]
session_token = credentials["SessionToken"]

# Step 4: Create a Bedrock client with temporary credentials
bedrock_client = boto3.client(
    "bedrock-runtime",
    region_name="ap-southeast-1",
    aws_access_key_id=access_key,
    aws_secret_access_key=secret_key,
    aws_session_token=session_token,
)

print(access_key)
print(secret_key)
print(session_token)

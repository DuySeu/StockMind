import boto3

from config import region, config, model


class Client:
    """Create a client to interact with the Bedrock API."""

    def __init__(self, system_prompt: str, inference_config: dict):
        self.inference_config = inference_config
        self.system_prompt = system_prompt
        self.model_id: str = model.CLAUDE_3_HAIKU
        self.session = boto3.Session(
            aws_access_key_id=config.AWS_ACCESS_KEY_ID,
            aws_secret_access_key=config.AWS_SECRET_ACCESS_KEY,
            region_name=region.AP_SOUTHEAST_1,
        )

    def _create_client(
        self,
        service_name: str = "bedrock-runtime",
    ):
        # Create client with the init credentials
        return self.session.client(
            service_name=service_name, region_name=region.AP_SOUTHEAST_1
        )

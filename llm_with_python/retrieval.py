import boto3
from botocore.exceptions import ClientError

from config import region, config, model


class Retrieval:
    """Retrieve context from Amazon Bedrock Knowledge Base"""

    def __init__(self, user_query: str, inference_config: dict):
        self.inference_config = inference_config
        self.model_id: str = model.CLAUDE_3_HAIKU
        self.user_query = user_query

    session = boto3.Session(
        aws_access_key_id=config.AWS_ACCESS_KEY_ID,
        aws_secret_access_key=config.AWS_SECRET_ACCESS_KEY,
        region_name=region.AP_SOUTHEAST_1,
    )

    def _create_client(
        self,
        service_name: str = "bedrock-agent-runtime",
    ):
        # Create client with the init credentials
        return self.session.client(
            service_name=service_name, region_name=region.AP_SOUTHEAST_1
        )

    # Define retrieve context
    def retrieve_context(
        self,
        kbId,
        numberOfResults=5,
        # overrideSearchType: str = 'SEMANTIC',
    ):
        # Initate bedrock client to retrieve context
        response = self._create_client().retrieve(
            retrievalQuery={"text": self.user_query},
            knowledgeBaseId=kbId,
            retrievalConfiguration={
                "vectorSearchConfiguration": {
                    "numberOfResults": numberOfResults,
                    # "overrideSearchType": overrideSearchType,
                }
            },
        )

        retrieval_results = response["retrievalResults"]
        return retrieval_results

    # Fetch contents from the response
    def get_contexts(self, retrieval_results):
        # Get context and sources
        contexts = []
        sources = []
        scores = []
        for i in range(len(retrieval_results)):
            contexts.append(retrieval_results[i]["content"]["text"])
            sources.append(retrieval_results[i]["location"]["s3Location"]["uri"])
            scores.append(retrieval_results[i]["score"])
        # Get unique sources
        sources = set(sources)
        sources = list(sources)
        return contexts, sources, scores

    # Get link for the URI
    def _parse_s3_uri(self, s3_uri):
        if s3_uri.startswith("s3://"):
            s3_uri = s3_uri[5:]
        else:
            raise ValueError("Invalid S3 Uri")
        bucket_name, object_key = s3_uri.split("/", 1)
        return bucket_name, object_key

    # Create the presigned for single link
    def _create_presigned_url(self, bucket_name, object_name, expiration=7200):
        # Generate a presigner uRL for the S3 Object
        s3_client = boto3.client("s3")
        try:
            response = s3_client.generate_presigned_url(
                "get_object",
                Params={"Bucket": bucket_name, "Key": object_name},
                ExpiresIn=expiration,
            )
        except ClientError as e:
            return None
        return response

    # Generate links for a list of uri
    def generate_presigned_urls(self, s3_uris, expiration=7200):
        presigned_urls = []
        for s3_uri in s3_uris:
            bucket_name, object_key = self._parse_s3_uri(s3_uri)
            presigned_url = self._create_presigned_url(
                bucket_name, object_key, expiration
            )
            if presigned_url:
                presigned_urls.append(presigned_url)
        return presigned_urls

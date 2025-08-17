import time
import boto3
import json
import logging

from config import region, config, model

# Setup logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)


class Tool_Call:
    """Invoke the model and get the response"""

    def __init__(
        self,
        system_prompt: str,
        tools: list,
        inference_config: dict,
        execute_tool: dict,
    ):
        self.system_prompt = system_prompt
        self.tools = tools
        self.inference_config = inference_config
        self.execute_tool = execute_tool

        self.model_id: str = model.CLAUDE_3_HAIKU
        self.session = boto3.Session(
            aws_access_key_id=config.AWS_ACCESS_KEY_ID,
            aws_secret_access_key=config.AWS_SECRET_ACCESS_KEY,
            region_name=region.AP_SOUTHEAST_1,
        )

    def _create_client(
        self,
        service_name: str = "bedrock-runtime",
    ) -> boto3.client:
        # Create client with the init credentials
        return self.session.client(
            service_name=service_name, region_name=region.AP_SOUTHEAST_1
        )

    def _prepare_payload(
        self,
        user_message: str,
        assistant_message: list = None,
        tool_id: str = None,
        tool_result: dict = None,
    ) -> dict:
        content = [{"type": "text", "text": user_message}]

        messages = [{"role": "user", "content": content}]

        if assistant_message and tool_id and tool_result is not None:
            messages.append({"role": "assistant", "content": assistant_message})
            messages.append(
                {
                    "role": "user",
                    "content": [
                        {
                            "type": "tool_result",
                            "tool_use_id": tool_id,
                            "content": [
                                {"type": "text", "text": json.dumps(tool_result)}
                            ],
                        }
                    ],
                }
            )

        body = {
            "anthropic_version": "bedrock-2023-05-31",
            **self.inference_config,
            "messages": messages,
            "system": self.system_prompt,
        }

        if len(self.tools) >= 1:
            body["tools"] = self.tools

        return {
            "body": json.dumps(body),
            "contentType": "application/json",
            "accept": "application/json",
            "modelId": "anthropic.claude-3-haiku-20240307-v1:0",
        }

    def invoke(self, message: str) -> dict:
        # Prepare payload
        payload = self._prepare_payload(user_message=message)
        # logger.info(f"Payload: {payload}")

        # Invoke the agent
        response = self._create_client().invoke_model(**payload)

        # Get response with information
        response_body = json.loads(response.get("body").read())
        logger.info(f"Response: {response_body}")

        tool_use_data = next(
            (
                item
                for item in response_body.get("content")
                if item.get("type") == "tool_use"
            ),
            None,
        )
        logger.info(f"Tool use data: {tool_use_data}")

        if tool_use_data.get("type") == "tool_use":
            tool_id = tool_use_data.get("id")
            tool_name = tool_use_data.get("name")
            tool_input = tool_use_data.get("input")

            tool_result = self.execute_tool(tool_name, tool_input)

            logger.info(f"Tool result: {tool_result}")

            final_payload = self._prepare_payload(
                user_message=message,
                assistant_message=response_body.get("content"),
                tool_id=tool_id,
                tool_result=tool_result,
            )

            tool_response = self._create_client().invoke_model(**final_payload)

            final_response = json.loads(tool_response.get("body").read())
        # else:
        #     final_response = response_body

        # Return response
        return {
            "answer": final_response["content"][0]["text"],
            "input_tokens": final_response["usage"]["input_tokens"],
            "output_tokens": final_response["usage"]["output_tokens"],
            "tool_data": tool_result,
        }

    def invoke_stream(self, message: str):
        payload = self._prepare_payload(user_message=message)
        response = self._create_client().invoke_model(**payload)
        response_body = json.loads(response.get("body").read())

        tool_use_data = next(
            (
                item
                for item in response_body.get("content")
                if item.get("type") == "tool_use"
            ),
            None,
        )
        # logger.info(f"Tool use data: {tool_use_data}")

        if tool_use_data.get("type") == "tool_use":
            tool_id = tool_use_data.get("id")
            tool_name = tool_use_data.get("name")
            tool_input = tool_use_data.get("input")

            tool_result = self.execute_tool(tool_name, tool_input)

            # logger.info(f"Tool result: {tool_result}")

            final_payload = self._prepare_payload(
                user_message=message,
                assistant_message=response_body.get("content"),
                tool_id=tool_id,
                tool_result=tool_result,
            )

            tool_response = self._create_client().invoke_model_with_response_stream(**final_payload)

            stream = tool_response.get("body")

                    # Yield message
            if stream:
                for event in stream:
                    # print(event)
                    chunk = event.get("chunk")
                    if chunk:
                        data = json.loads(chunk.get("bytes").decode())
                        logger.info(data)

        # Yield message
        # if stream:
        #     for event in stream:
        #         print(event)
        #         chunk = event.get("chunk")
        #         if chunk:
        #             data = json.loads(chunk.get("bytes").decode())

        #             # Yield normal text
        #             if (
        #                 data["type"] == "content_block_delta"
        #                 and data["delta"]["type"] == "text_delta"
        #             ):
        #                 text = data["delta"].get("text")
        #                 if text:
        #                     yield text
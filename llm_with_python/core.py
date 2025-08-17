import boto3
import json

from config import region, config, model

from typing import List, Dict, Any, Union, Generator


# Create class LLMWorker
class LLMWorker:
    """Invoke the model and get the response"""

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

    def _prepare_payload(self, message: dict) -> dict:
        content = [{"type": "text", "text": message.content}]

        if message.file is not None:
            content.append(
                {
                    "type": "image",
                    "source": {
                        "data": message.file.base64,
                        "media_type": message.file.mediaType,
                        "type": "base64",
                    },
                }
            )

        messages = [{"role": "user", "content": content}]

        body = {
            "anthropic_version": "bedrock-2023-05-31",
            **self.inference_config,
            "messages": messages,
            "system": self.system_prompt,
            "tools": self.tools,
        }

        return {
            "body": json.dumps(body),
            "contentType": "application/json",
            "accept": "application/json",
            "modelId": self.model_id,
        }

    def invoke(self, message: Dict[str, Any]) -> Dict:
        # Prepare payload
        payload = self._prepare_payload(message=message)

        # Invoke the agent
        response = self._create_client().invoke_model(**payload)

        # Get response with infor
        response_body = json.loads(response.get("body").read())

        # Return response
        return {
            "answer": response_body["content"][0]["text"],
            "input_tokens": response_body["usage"]["input_tokens"],
            "output_tokens": response_body["usage"]["output_tokens"],
        }

    def invoke_stream(self, message: Dict[str, Any]) -> Generator:
        payload = self._prepare_payload(message=message)
        response = self._create_client().invoke_model_with_response_stream(**payload)
        stream = response.get("body")

        # Yield message
        if stream:
            for event in stream:
                print(event)
                chunk = event.get("chunk")
                if chunk:
                    data = json.loads(chunk.get("bytes").decode())

                    # Yield normal text
                    if (
                        data["type"] == "content_block_delta"
                        and data["delta"]["type"] == "text_delta"
                    ):
                        text = data["delta"].get("text")
                        if text:
                            yield text


class Agent(LLMWorker):
    def __init__(self, max_history: int = 20, **kwargs):
        super().__init__(**kwargs)
        self.max_history = max_history
        self.chat_history = []
        self.tools = []
        self.tool_registry = {}

    def _prepare_payload(
        self,
        message: Union[str, List[Dict[str, Any]]],
    ) -> Dict[str, Any]:
        if isinstance(message, str):
            self.chat_history.append(
                {"role": "user", "content": [{"type": "text", "text": message}]}
            )
        else:
            message = message

        body = {
            "anthropic_version": "bedrock-2023-05-31",
            **self.inference_config,
            "messages": self.chat_history,
            "system": self.system_prompt,
        }

        if len(self.tools) >= 1:
            body["tools"] = self.tools

        return {
            "body": json.dumps(body),
            "contentType": "application/json",
            "accept": "application/json",
            "modelId": self.model_id,
        }

    def _manage_history(self) -> None:
        if len(self.chat_history) >= self.max_history:
            self.chat_history = self.chat_history[4:]

    def add_tool(self, tool: Dict[str, Any], func) -> None:
        self.tools.append(tool)
        self.tool_registry[tool["name"]] = func

    def invoke(self, message: str) -> Dict:
        response = self._create_client().invoke(
            **self._prepare_payload(message=message)
        )
        response_body = json.loads(response.get("body").read())
        return {
            "answer": response_body["content"][0]["text"],
            "input_tokens": response_body["usage"]["input_tokens"],
            "output_tokens": response_body["usage"]["output_tokens"],
        }

    def invoke_stream(self, message: str):
        payload = self._prepare_payload(message=message)
        while True:
            response = self._create_client().invoke_model_with_response_stream(
                **payload
            )
            stream = response.get("body")

            # Flag for using tool
            tool_used = False
            tool_name = None
            tool_id = None
            tool_request = {}

            partial_json_string = ""
            bot_response = ""

            # Yield message
            if stream:
                for event in stream:
                    chunk = event.get("chunk")
                    if chunk:
                        data = json.loads(chunk.get("bytes").decode())

                        # Yield normal text
                        if (
                            data["type"] == "content_block_delta"
                            and data["delta"]["type"] == "text_delta"
                        ):
                            text = data["delta"].get("text")
                            if text:
                                bot_response += text
                                yield text

                        # Tool decision
                        elif (
                            data["type"] == "content_block_start"
                            and data["content_block"]["type"] == "tool_use"
                        ):
                            tool_used = True
                            tool_id = data["content_block"]["id"]
                            tool_name = data["content_block"]["name"]

                        # Receive tool schema
                        elif (
                            data["type"] == "content_block_delta"
                            and data["delta"]["type"] == "input_json_delta"
                        ):
                            if "partial_json" in data["delta"]:
                                partial_json_string += data["delta"]["partial_json"]
                                try:
                                    parsed_json = json.loads(partial_json_string)
                                    tool_request.update(parsed_json)
                                except json.JSONDecodeError:
                                    pass
                # Execute tool
                if tool_used and tool_name in self.tool_registry:
                    # Save assistant message with tool use
                    self.chat_history.append(
                        {
                            "role": "assistant",
                            "content": [
                                {"type": "text", "text": bot_response},
                                {
                                    "type": "tool_use",
                                    "name": tool_name,
                                    "id": tool_id,
                                    "input": tool_request,
                                },
                            ],
                        }
                    )

                    # Execute tool
                    tool_result = self.tool_registry[tool_name](**tool_request)

                    # Save tool result
                    self.chat_history.append(
                        {
                            "role": "user",
                            "content": [
                                {
                                    "type": "tool_result",
                                    "tool_use_id": tool_id,
                                    "content": [
                                        {"type": "text", "text": str(tool_result)}
                                    ],
                                }
                            ],
                        }
                    )

                    # Prepare new payload
                    payload = self._prepare_payload(message=self.chat_history)
                else:
                    if bot_response:
                        self.chat_history.append(
                            {
                                "role": "assistant",
                                "content": [{"type": "text", "text": bot_response}],
                            }
                        )

                        # Clean history
                        self._manage_history()
                        # Then break
                        break

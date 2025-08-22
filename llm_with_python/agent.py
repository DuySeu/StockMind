from dotenv import load_dotenv
from openai import OpenAI
import os

# Load environment variables from .env file
load_dotenv()

OPENROUTER_API_KEY = os.getenv("OPENROUTER_API_KEY")

# Model ID in OpenRouter support function calling and free (https://openrouter.ai/models)
GLM_4_5_AIR = "z-ai/glm-4.5-air:free"
QWEN3_CODE = "qwen/qwen3-coder:free"
QWEN3_4B = "qwen/qwen3-4b:free"
QWEN3_235B = "qwen/qwen3-235b-a22b:free"
KIMI_K2 = "moonshotai/kimi-k2:free"
MISTRAL_SMALL = "mistralai/mistral-small-3.2-24b-instruct:free"
DEVTRAL_SMALL = "mistralai/devstral-small-2505:free"
DEEPSEEK_V3 = "deepseek/deepseek-chat-v3-0324:free"


class Agent:
    """
    This class is used to create an agent that can invoke a model and return a response.
    It can also invoke a model in a streaming manner.
    It can also be used to invoke a model with a tool.
    It can also be used to invoke a model with a tool in a streaming response.

    This Agent using OpenAI's API. To call API model in OpenRouter, easy to change model to test every free model.
    """

    def __init__(
        self,
        system_prompt: str,
        inference_config: dict,
        tools: list | None = None,
    ):
        self.system_prompt = system_prompt
        self.inference_config = inference_config
        self.model_id = GLM_4_5_AIR
        self.tools = tools

    def _create_client(self):
        return OpenAI(
            base_url="https://openrouter.ai/api/v1",
            api_key=OPENROUTER_API_KEY,
        )

    def _prepare_payload(self, message: str | list[dict], stream: bool = False) -> dict:
        if isinstance(message, str):
            messages = [{"role": "user", "content": message}]
        else:
            messages = message

        payload = {
            "model": self.model_id,
            "messages": messages,
            **self.inference_config,
            "stream": stream,
        }

        # Configure tool calling if tools exist
        if self.tools:
            payload["tools"] = self.tools
            payload["tool_choice"] = "auto"

        return payload

    def invoke_model(self, message: str | list[dict], stream: bool = False) -> str:
        payload = self._prepare_payload(message=message, stream=stream)
        response = self._create_client().chat.completions.create(**payload)

        if stream:
            for chunk in response:
                if (
                    hasattr(chunk.choices[0], "delta")
                    and chunk.choices[0].delta.content
                ):
                    piece = chunk.choices[0].delta.content
                    yield piece
        else:
            return response.choices[0].message.content

    def invoke_tool(self, message: str | list[dict]) -> str:
        payload = self._prepare_payload(message=message, stream=False)
        response = self._create_client().chat.completions.create(**payload)
        return response.choices[0]

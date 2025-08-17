import os
from dotenv import load_dotenv
from pydantic import BaseModel
from typing import Optional, Dict

# Load dot environment
load_dotenv()


# Setup class configs
class Config:
    AWS_ACCESS_KEY_ID = os.getenv("AWS_ACCESS_KEY_ID") or ""
    AWS_SECRET_ACCESS_KEY = os.getenv("AWS_SECRET_ACCESS_KEY") or ""
    KB_ID = os.getenv("KB_ID") or ""
    GROQ_API = os.getenv("GROQ_API") or ""


class ModelId:
    CLAUDE_3_HAIKU = "anthropic.claude-3-haiku-20240307-v1:0"
    CLAUDE_3_5_HAIKU = "us.anthropic.claude-3-5-haiku-20241022-v1:0"
    CLAUDE_3_5_SONNET_V1 = "anthropic.claude-3-5-sonnet-20240620-v1:0"
    CLAUDE_3_5_SONNET_V2 = "us.anthropic.claude-3-5-sonnet-20241022-v2:0"

    GROQ_LLAMA_3_3_70B_VERSATILE = "llama-3.3-70b-versatile"
    GROQ_LLAMA_3_1_8B_INSTANT = "llama-3.1-8b-instant"
    GROQ_LLAMA_3_GUARD_8B = "llama-guard-3-8b"
    GROQ_LLAMA_3_70B_8192 = "llama3-70b-8192"
    GROQ_LLAMA_3_8B_8192 = "llama3-8b-8192"
    GROQ_MIXTRAL_8X7B_32768 = "mixtral-8x7b-32768"


class Region:
    AP_SOUTHEAST_1 = "ap-southeast-1"


message = [
    {
        "content": "string",
        "file": {
            "base64": "string",
            "fileName": "string",
            "mediaType": "string",
            "isText": "boolean",
        },
    }
]


config = Config()
model = ModelId()
region = Region()

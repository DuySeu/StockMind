import time
import logging

from core import LLMWorker
from retrieval import Retrieval
from simple_prompt import text_to_query_prompt, prompt, prompt_for_web_search
from tool_call import Tool_Call
from tool_schema import get_realtime_tool, generate_chart_tool, generate_OTP_tool
from execute_chart_tool import execute_chart
from execute_OTP_tool import execute_OTP
from execute_web_search_tool import execute_web_search

# Setup logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)


def main():
    print(time.strftime("%H:%M:%S", time.localtime()))
    request = "I want to subscribe to 4G packet."
    # request = "Vietnamobile was established in which year?"
    config = {"max_tokens": 2048, "temperature": 0.7, "top_k": 250}

    retrieve_worker = Retrieval(
        user_query=request,
        inference_config=config,
    )

    retrieval_results = retrieve_worker.retrieve_context(
        kbId="YPHXAPJE6I",
        numberOfResults=3,
    )
    # logging.info(f"Retrieved result: {retrieval_results}")

    contexts, sources, scores = retrieve_worker.get_contexts(retrieval_results)
    # link = retrieve_worker.generate_presigned_urls(sources)
    # logging.info(f"Presigned Urls: {contexts}")

    test = Tool_Call(
        system_prompt=prompt,
        tools=generate_OTP_tool,
        inference_config=config,
        execute_tool=execute_OTP,
    )

    result = test.invoke_stream(request)

    # print(f"Result: {result}")
    # print(result.get("answer"))


if __name__ == "__main__":
    main()

import time
import logging
import json

from simple_prompt import prompt
from agent import Agent
from tool_schema import get_stock_price_tool
from execute_stockprice_tool import get_stock_price

# Setup logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)

MODEL_CONFIG = {"max_tokens": 2048, "temperature": 0.7}


def process_tool_call(response, input_list):
    if response.finish_reason == "tool_calls":
        tool_calls = response.message.tool_calls

        if tool_calls:
            input_list.append(
                {
                    "role": "assistant",
                    "tool_calls": tool_calls,
                }
            )

            # Execute the first tool call
            tool_call = tool_calls[0]
            function_name = tool_call.function.name
            function_args = json.loads(tool_call.function.arguments)

            # Execute the tool based on function name
            if function_name == "get_stock_price":
                stock_code = function_args.get("stock_code")
                if stock_code:
                    try:
                        result = get_stock_price(stock_code)

                        input_list.append(
                            {
                                "tool_call_id": tool_call.id,
                                "role": "tool",
                                "content": str(result),
                            }
                        )
                    except Exception as e:
                        input_list.append(
                            {
                                "tool_call_id": tool_call.id,
                                "role": "tool",
                                "content": f"Error: {str(e)}",
                            }
                        )
    return input_list


def main():
    print(time.strftime("%H:%M:%S", time.localtime()))
    request = "What is the stock price of VNM?"
    input_list = [
        {"role": "user", "content": request},
    ]

    agent = Agent(
        system_prompt=prompt,
        inference_config=MODEL_CONFIG,
        tools=get_stock_price_tool,
    )

    response = agent.invoke_tool(input_list)
    print(f"Response: {response.message.content}")

    full_input_list = process_tool_call(response, input_list)

    final_result = agent.invoke_model(full_input_list, stream=True)
    for piece in final_result:
        print(piece, end="", flush=True)
    print()


if __name__ == "__main__":
    main()

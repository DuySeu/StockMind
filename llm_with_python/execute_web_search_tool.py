from duckduckgo_search import DDGS
import logging

# Setup logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)


def execute_web_search(tool_name, tool_input):
    # Execute web search tool
    if tool_name is None:
        return None

    ddgs = DDGS()

    query = tool_input.get("query")

    # 1. Resposta de Chat IA
    chat_response = ddgs.chat(query)
    print(chat_response if chat_response else "Sem resposta do Chat IA.")

    # 2. Pesquisa de Texto
    text_results = ddgs.text(query, max_results=5)
    if text_results:
        for result in text_results:
            print(f"Title: {result['title']}")
            print(f"Link: {result['href']}")
            print(f"Content: {result['body']}")
            print("-" * 50)

    return text_results, chat_response


# if __name__ == "__main__":
#     query = "Vietnamobile thành lập năm bao nhiêu?"
#     result = web_search_tool(query)
#     print(result)

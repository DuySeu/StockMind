# Define the tool schema structure
get_realtime_tool = [
    {
        "name": "get_time",
        "description": "Get the current time in a given location",
        "input_schema": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "The city and state, e.g. San Francisco, CA",
                },
                "unit": {
                    "type": "string",
                    "enum": ["%H:%M:%S"],
                    "description": 'The unit of hour is "%H:%M:%S"',
                },
            },
            "required": ["location"],
        },
    }
]

generate_chart_tool = [
    {
        "name": "generate_graph_data",
        "description": "Generate structured JSON data for creating financial charts and graphs.",
        "input_schema": {
            "type": "object",
            "properties": {
                "chartType": {
                    "type": "string",
                    "enum": ["bar", "multiBar", "line", "pie", "area", "stackedArea"],
                    "description": "The type of chart to generate",
                },
                "config": {
                    "type": "object",
                    "properties": {
                        "title": {"type": "string"},
                        "description": {"type": "string"},
                        "totalLabel": {"type": "string"},
                        "xAxisKey": {"type": "string"},
                        "yAxisKey": {"type": "string"},
                    },
                    "required": ["title", "description"],
                },
                "data": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "additionalProperties": True,
                    },
                },
                "chartConfig": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "object",
                        "properties": {
                            "label": {"type": "string"},
                            "stacked": {"type": "boolean"},
                        },
                        "required": ["label"],
                    },
                },
            },
            "required": ["chartType", "config", "data", "chartConfig"],
        },
    }
]

web_search_tool = [
    {
        "name": "web_search",
        "description": "Search the web for Vietnamobile static information.",
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "The search query to look up on the web.",
                },
                "numResults": {
                    "type": "integer",
                    "description": "The number of search results to retrieve.",
                    "default": 3,
                },
            },
            "required": ["query"],
        },
    }
]

generate_OTP_tool = [
    {
        "name": "generate_OTP",
        "description": "Generate a one-time password (OTP) for subscribe or unsubscribe Internet Data Packet.",
        "input_schema": {
            "type": "object",
            "properties": {
                "internetDataPacket": {
                    "type": "string",
                    "description": "The name of the Internet Data Packet, e.g. 3G, 4G, 5G.",
                },
            },
            "required": ["OTP", "internetDataPacket"],
        },
    }
]

get_stock_price_tool = [
    {
        "type": "function",
        "function": {
            "name": "get_stock_price",
            "description": "Get the current stock price of a given stock",
            "parameters": {
                "type": "object",
                "properties": {
                    "stock_code": {
                        "type": "string",
                        "description": "The stock code, e.g. VNM",
                    },
                },
                "required": ["stock_code"],
            },
        },
    }
]

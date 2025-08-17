import logging

# Setup logging
logger = logging.getLogger(__name__)
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)


def execute_chart(tool_name, tool_input):
    if tool_name is None:
        return None

    chartData = tool_input

    # logger.info(f"Preprocessing for pie data {chartData}")

    # Transform data format for pie chart
    for item in chartData.get("data", []):
        # Find the key that represents the segment (category/label)
        segment_key = next(
            (
                key
                for key in item.keys()
                if key.lower() in ["sex", "category", "label", "name", "type"]
            ),
            None,
        )
        # Find the key that represents the value (numeric data)
        value_key = next(
            (
                key
                for key in item.keys()
                if key.lower() in ["expenses", "value", "amount", "count"]
            ),
            None,
        )

        if segment_key and value_key:
            item["segment"] = item.pop(segment_key)
            item["value"] = item.pop(value_key)

    processed_chart_config = {}
    for index, (key, config) in enumerate(chartData.get("chartConfig", {}).items()):
        processed_chart_config[key] = {
            **config,
            "color": f"hsl(var(--chart-{index + 1}))",
        }

    # Return updated chart data with processed config
    return {**chartData, "chartConfig": processed_chart_config}

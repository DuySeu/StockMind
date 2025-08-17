{
    "chartType": "pie",
    "config": {
        "title": "Insurance Expenses by Sex",
        "description": "Percentage breakdown of insurance expenses between males and females",
        "totalLabel": "Total Insurance Expenses",
    },
    "data": [
        {"sex": "male", "expenses": 63987.53},
        {"sex": "female", "expenses": 114401.96},
    ],
    "chartConfig": {"male": {"label": "Male"}, "female": {"label": "Female"}},
}

{
    "chartType": "pie",
    "config": {
        "title": "Insurance Expenses by Sex",
        "description": "Percentage breakdown of insurance expenses between males and females",
        "totalLabel": "Total Expenses",
    },
    "data": [
        {"segment": "Female", "value": 233675.81},
        {"segment": "Male", "value": 227401.18},
    ],
    "chartConfig": {
        "Female": {"label": "Female", "color": "hsl(var(--chart-1))"},
        "Male": {"label": "Male", "color": "hsl(var(--chart-2))"},
    },
}

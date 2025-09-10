import requests
import os
import asyncpg
import asyncio
import fastapi
import json

from logsim import CustomLogger
from fastapi import APIRouter, BackgroundTasks
from fastapi.responses import JSONResponse, Response

from dependencies.main import get_db, get_request_id
from core.database import execute_query
from core.s3_client import get_s3_client

# Setup logger
logger = CustomLogger()

# Setup router
router = APIRouter(prefix="/flow", tags=["flow"])

S3_BUCKET_NAME = os.getenv("S3_BUCKET_NAME")
TIME_TO_MONITOR_FLOW = int(os.getenv("TIME_TO_MONITOR_FLOW", 5))

active_monitoring_tasks: dict[str, asyncio.Task] = {}


async def create_flow(transaction_id: str) -> str | None:
    """Create a flow"""
    try:
        response = requests.post(
            "http://pinazu:8081/v1/flows",
            json={
                "name": transaction_id,
                "parameters_schema": {
                    "job_id": transaction_id,
                },
                "engine": "process",
                "code_location": "s3://test-bucket-302010997939-ap-southeast-1/flows/mb-idp.py",
                "entrypoint": "python",
                "tags": ["mb", "idp"],
            },
            timeout=30,
        )
        flow_response = response.json()
        if flow_response.get("id"):
            return flow_response.get("id")
        else:
            logger.error(f"Failed to create flow: {flow_response}")
            return None
    except Exception as e:
        logger.error(f"Failed to create flow: {str(e)}")


@router.post("/{flow_id}/execute", status_code=204)
async def execute_flow(
    flow_id: str,
    transaction_id: str,
    background_tasks: BackgroundTasks,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Execute a flow"""

    update_document_status_query = """
        UPDATE documents
        SET status = $1, updated_at = NOW()
        WHERE flow_id = $2
    """

    try:
        response = requests.post(
            f"http://pinazu:8081/v1/flows/{flow_id}/execute",
            json={
                "parameters": {
                    "job_id": transaction_id,
                },
            },
            timeout=30,
        )
        execute_flow_info = response.json()
        logger.info(f"Execute flow info: {execute_flow_info}")
        await execute_query(
            query=update_document_status_query,
            params=(execute_flow_info.get("status"), flow_id),
            request_id=request_id,
            connection=conn,
        )

        flow_run_id = execute_flow_info.get("flow_run_id")

        if flow_run_id:
            # Create and start the background monitoring task
            background_tasks.add_task(
                monitor_flow_status_background,
                flow_id,
                flow_run_id,
                request_id,
                transaction_id,
            )

            logger.info(f"Started background monitoring for flow run: {flow_run_id}")
        else:
            logger.warning("No flow_run_id received, skipping background monitoring")

        return Response(status_code=204)
    except Exception as e:
        logger.error(f"Failed to execute flow: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to execute flow: {str(e)}"},
            status_code=500,
        )


async def monitor_flow_status_background(
    flow_id: str, flow_run_id: str, request_id: str, transaction_id: str
):
    """
    Background task to monitor flow status every specific seconds
    until status becomes 'failed' or 'success'
    """

    update_document_status_query = """
        UPDATE documents
        SET status = $1, updated_at = NOW()
        WHERE flow_id = $2
    """

    update_document_quality_query = """
        UPDATE documents
        SET quality = $1, updated_at = NOW()
        WHERE flow_id = $2 AND name = $3
    """

    try:
        while True:
            try:
                response = requests.get(
                    f"http://pinazu:8081/v1/flows/{flow_run_id}/status",
                    timeout=30,
                )

                if response.status_code == 200:
                    status_data = response.json()
                    current_status = status_data.get("status")

                    logger.info(f"Flow {flow_run_id} status: {current_status}")

                    # Update document status in database
                    await execute_query(
                        query=update_document_status_query,
                        params=(current_status, flow_id),
                        request_id=request_id,
                    )

                    # Check if flow is completed (failed or success)
                    if current_status.upper() in ["FAILED", "SUCCESS"]:
                        logger.info(
                            f"Flow {flow_run_id} completed with status: {current_status}"
                        )

                        # Only try to update quality if the flow was successful
                        if current_status.upper() == "SUCCESS":
                            try:
                                # Update document quality in database
                                quality_s3_key = (
                                    f"{transaction_id}/quality_result/quality.json"
                                )
                                quality_file = get_s3_client().get_object(
                                    Bucket=S3_BUCKET_NAME,
                                    Key=quality_s3_key,
                                )
                                quality_result_json = json.load(quality_file["Body"])
                                quality_result_content = quality_result_json["files"]
                                for (
                                    file_key,
                                    file_value,
                                ) in quality_result_content.items():
                                    pages = file_value["pages"]
                                    overall_quality = 0
                                    for page in pages:
                                        overall_quality += page["quality_metrics"][
                                            "overall_quality"
                                        ]
                                    overall_quality = overall_quality / len(pages)

                                    if overall_quality > 80:
                                        quality = "good"
                                    elif overall_quality > 60:
                                        quality = "fair"
                                    else:
                                        quality = "poor"

                                    logger.info(f"Overall quality: {overall_quality}")
                                    logger.info(f"File key: {file_key}")
                                    await execute_query(
                                        query=update_document_quality_query,
                                        params=(quality, flow_id, file_key),
                                        request_id=request_id,
                                    )
                            except Exception as e:
                                logger.error(
                                    f"Error processing quality results: {str(e)}"
                                )
                        break

                    # Wait before next check
                    await asyncio.sleep(TIME_TO_MONITOR_FLOW)

                else:
                    logger.error(
                        f"Failed to get flow status. Status code: {response.status_code}"
                    )
                    await asyncio.sleep(TIME_TO_MONITOR_FLOW)

            except requests.exceptions.RequestException as e:
                logger.error(f"Request error monitoring flow {flow_run_id}: {str(e)}")
                await asyncio.sleep(TIME_TO_MONITOR_FLOW)
            except Exception as e:
                logger.error(f"Error monitoring flow {flow_run_id}: {str(e)}")
                await asyncio.sleep(TIME_TO_MONITOR_FLOW)

    except Exception as e:
        logger.error(f"Fatal error in flow monitoring for {flow_run_id}: {str(e)}")
    finally:
        # Clean up monitoring task
        if flow_run_id in active_monitoring_tasks:
            del active_monitoring_tasks[flow_run_id]
        logger.info(f"Stopped monitoring flow {flow_run_id}")

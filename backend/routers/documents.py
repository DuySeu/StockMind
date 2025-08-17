import uuid
import asyncpg
import fastapi
import random
import os
import json
import io
from typing import Any

from datetime import datetime
from pydantic import BaseModel
from logsim import CustomLogger
from fastapi import APIRouter, UploadFile, File, Form, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse

from dependencies.main import get_db, get_request_id, get_s3
from core.database import execute_query
from routers.flow import create_flow, execute_flow

# Setup logger
logger = CustomLogger()

# Setup router
router = APIRouter(prefix="/documents", tags=["documents"])

S3_BUCKET_NAME = os.getenv("S3_BUCKET_NAME")


# Model for Documents
class DocumentStatus(BaseModel):
    status: str
    flow_id: str


@router.get("")
async def get_documents(
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Get all documents or a limited number"""

    query = """
        SELECT d.id, d.name, d.transaction_id, d.template_id, d.status, d.quality, d.flow_id, t.name as template_name, d.created_at
        FROM documents d
        LEFT JOIN templates t ON d.template_id = t.id
        ORDER BY d.updated_at DESC
    """
    try:
        documents = await execute_query(
            query=query,
            params=(),
            request_id=request_id,
            connection=conn,
        )

        return JSONResponse(content=documents, status_code=200)
    except Exception as e:
        return JSONResponse(
            content={"message": f"Error in getting all documents: {str(e)}"},
            status_code=500,
        )


@router.get("/{id}")
async def get_document_in_s3(
    id: str,
    s3_client: Any = fastapi.Depends(get_s3),
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Get a specific document by ID"""

    # Validate S3 bucket name
    if not S3_BUCKET_NAME:
        raise HTTPException(
            status_code=500,
            detail="S3 bucket name not set",
        )

    get_document_query = """
        SELECT id, name, transaction_id
        FROM documents
        WHERE id = $1
    """

    try:
        if not id:
            logger.error("The document id is empty.")
            return JSONResponse(
                content={"message": f"Failed to retrieve document with id {id}."},
                status_code=400,
            )

        document = await execute_query(
            query=get_document_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not document:
            logger.error(f"The document id {id} does not exist")
            return JSONResponse(
                content={"message": f"Document with ID {id} does not exist"},
                status_code=404,
            )

        raw_file_s3_key = (
            f"{document[0]['transaction_id']}/raw_file/{document[0]['name']}"
        )
        quality_s3_key = f"{document[0]['transaction_id']}/quality_result/quality.json"
        processed_result_s3_key = (
            f"{document[0]['transaction_id']}/processed_results/results.json"
        )

        # Get file from S3
        try:
            # Get raw file from S3
            raw_file = s3_client.generate_presigned_url(
                ClientMethod="get_object",
                Params={"Bucket": S3_BUCKET_NAME, "Key": raw_file_s3_key},
                ExpiresIn=300,
            )

            # Get quality result from S3
            quality_result = s3_client.get_object(
                Bucket=S3_BUCKET_NAME, Key=quality_s3_key
            )
            quality_result_json = json.load(quality_result["Body"])
            quality_result_content = quality_result_json["files"][document[0]["name"]][
                "pages"
            ]
            logger.info(f"Quality result: {quality_result_content}")

            # Get processed result from S3
            processed_result = s3_client.get_object(
                Bucket=S3_BUCKET_NAME, Key=processed_result_s3_key
            )
            processed_result_json = json.load(processed_result["Body"])
            processed_result_content = next(
                (
                    content
                    for content in processed_result_json
                    if content["file_name"] == document[0]["name"]
                ),
                None,
            )

            file_detail = {
                "raw_file": raw_file,
                "quality_result": quality_result_content,
                "processed_result": processed_result_content,
            }
        except Exception as e:
            logger.error(f"Failed to retrieve file from S3: {str(e)}")
            file_detail = None

        return JSONResponse(content=file_detail, status_code=200)

    except Exception as e:
        logger.error(f"Failed to retrieve document with id {id}: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to retrieve document with ID {id}"},
            status_code=500,
        )


@router.post("", status_code=201)
async def create_document(
    files: list[UploadFile] = File(...),
    template_ids: str = Form(...),
    s3_client: Any = fastapi.Depends(get_s3),
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
    background_tasks: BackgroundTasks = BackgroundTasks(),
):
    """API to create new documents"""

    # Validate S3 bucket name
    if not S3_BUCKET_NAME:
        raise HTTPException(
            status_code=500,
            detail="S3 bucket name not set",
        )

    check_existing_template_id_query = """
        SELECT id
        FROM templates
        WHERE id = $1
    """

    check_existing_transaction_id_query = """
        SELECT transaction_id
        FROM documents
        WHERE transaction_id = $1
    """

    create_document_query = """
        INSERT INTO documents (id, name, transaction_id, template_id, status, quality, flow_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    """

    get_template_query = """
        SELECT t.id, t.name, t.description, t.field, t.prompt, t.created_at, t.updated_at, COALESCE(
            json_agg(
                json_build_object(
                    'id', r.id,
                    'name', r.name,
                    'description', r.description,
                    'rule_type', r.rule_type,
                    'condition', r.condition,
                    'action', r.action,
                    'created_at', r.created_at,
                    'updated_at', r.updated_at
                )
            ) FILTER (WHERE r.id IS NOT NULL),
            '[]'::json
        ) as rule_list
        FROM templates t
        LEFT JOIN template_rule_mapping trm ON t.id = trm.template_id
        LEFT JOIN rules r ON trm.rule_id = r.id
        WHERE t.id = $1
        GROUP BY t.id, t.name, t.description, t.field, t.prompt, t.created_at, t.updated_at
    """

    try:
        template_ids_list = [tid.strip() for tid in template_ids.split(",")]
        if not files or not template_ids_list:
            logger.error("Document must have files and template_ids.")
            raise HTTPException(
                status_code=400,
                detail="Files and template_ids are required.",
            )

        if len(files) != len(template_ids_list):
            logger.error("Number of files and template_ids must match.")
            raise HTTPException(
                status_code=400,
                detail="Number of files and template_ids must match.",
            )

        for template_id in template_ids_list:
            existing_template = await execute_query(
                query=check_existing_template_id_query,
                params=(template_id,),
                request_id=request_id,
                connection=conn,
            )

            if not existing_template:
                logger.error(f"Template with id {template_id} does not exist")
                raise HTTPException(
                    status_code=400,
                    detail=f"Template with id {template_id} does not exist",
                )

        # Default values
        format_date = datetime.now().strftime("%y%m")

        while True:
            transaction_number = "".join([str(random.randint(0, 9)) for _ in range(5)])
            transaction_id = f"XDP{format_date}{transaction_number}"

            # Check if transaction ID already exists
            existing_transaction_id = await execute_query(
                query=check_existing_transaction_id_query,
                params=(transaction_id,),
                request_id=request_id,
                connection=conn,
            )

            if not existing_transaction_id:
                # Found unique transaction ID
                break

        status = "pending"
        quality = None
        # Create a flow
        flow_id = await create_flow(transaction_id)

        # Determine processing type
        if len(files) == 1:
            processing_type = "single"
        elif len(set(template_ids_list)) == 1:
            processing_type = "combined"
        else:
            processing_type = "multiple"

        # Create a template.json for the document
        template_json = []

        for i, file in enumerate(files):
            logger.info(f"Uploading file: {file}")

            # Generate a unique ID for the document
            id = str(uuid.uuid4())
            name = file.filename
            template_id = template_ids_list[i]

            template = await execute_query(
                query=get_template_query,
                params=(template_id,),
                request_id=request_id,
                connection=conn,
            )
            template[0]["rule_list"] = json.loads(template[0]["rule_list"])
            template_json.append(template[0])

            # Read file content
            file_content = file.file.read()

            # Upload file to S3 with proper metadata
            s3_client.put_object(
                Bucket=S3_BUCKET_NAME,
                Key=f"{transaction_id}/raw_file/{name}",
                Body=file_content,
                ContentType=file.content_type or "application/octet-stream",
                Metadata={"template_id": template_id},
            )

            # Create document in database
            await execute_query(
                query=create_document_query,
                params=(
                    id,
                    name,
                    transaction_id,
                    template_id,
                    status,
                    quality,
                    flow_id,
                ),
                request_id=request_id,
                connection=conn,
            )

        # Convert template_json to JSON string
        template_json_string = json.dumps(template_json, indent=2, default=str)
        # Create a file-like object in memory
        template_file = io.BytesIO(template_json_string.encode("utf-8"))

        # Upload template.json to S3
        s3_client.put_object(
            Bucket=S3_BUCKET_NAME,
            Key=f"{transaction_id}/template/template.json",
            Body=template_file,
            ContentType="application/json",
            Metadata={"processing_type": processing_type},
        )

        try:
            await execute_flow(
                flow_id=flow_id,
                transaction_id=transaction_id,
                background_tasks=background_tasks,
                conn=conn,
                request_id=request_id,
            )
            logger.info(f"Successfully executed flow: {flow_id}")
        except Exception as e:
            logger.error(f"Failed to execute flow: {str(e)}")

        return JSONResponse(
            content={"message": "Successfully created document"},
            status_code=201,
        )

    except HTTPException:
        raise

    except Exception as e:
        logger.error(f"Failed to create document: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to create document: {str(e)}"},
            status_code=500,
        )


@router.put("/{id}", status_code=204)
async def update_document_status(
    id: str,
    request: DocumentStatus,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Edit a document by ID"""

    update_document_query = """
        UPDATE documents
        SET status = $1, flow_id = $2, updated_at = NOW()
        WHERE id = $3
    """

    try:
        if not id:
            logger.error("Error in editing document: document id must not be empty")
            return JSONResponse(
                content={
                    "message": "Failed to edit document. The document ID is empty."
                },
                status_code=400,
            )

        status = request.status
        flow_id = request.flow_id

        if not status or not flow_id:
            logger.error("The document must include the name.")
            return JSONResponse(
                content={"message": "The document must have status and flow_id"},
                status_code=400,
            )

        await execute_query(
            query=update_document_query,
            params=(status, flow_id, id),
            request_id=request_id,
            connection=conn,
        )

        return JSONResponse(
            content={},
            status_code=204,
        )

    except Exception as e:
        logger.exception(
            f"Unexpected error occurred while editing document {id}: {str(e)}"
        )
        return JSONResponse(
            content={"message": f"An unexpected error occurred: {str(e)}"},
            status_code=500,
        )


@router.delete("/{id}", status_code=204)
async def delete_document(
    id: str,
    s3_client: Any = fastapi.Depends(get_s3),
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Delete a document by ID"""

    delete_document_query = """
        DELETE FROM documents
        WHERE id = $1
    """
    check_document_query = """
        SELECT id, transaction_id, name
        FROM documents
        WHERE id = $1
    """
    try:
        if not id:
            logger.error("Error in deleting document: document id must not be empty")
            return JSONResponse(
                content={
                    "message": "Failed to delete document. The document ID is empty."
                },
                status_code=400,
            )

        # Check if document exists
        check_document = await execute_query(
            query=check_document_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not check_document:
            logger.error(f"The document id {id} does not exist")
            return JSONResponse(
                content={"message": f"Document with ID {id} does not exist"},
                status_code=404,
            )

        # Delete file from S3
        s3_key = f"{check_document[0]['transaction_id']}/raw_file/{check_document[0]['name']}"
        s3_client.delete_object(Bucket=S3_BUCKET_NAME, Key=s3_key)
        logger.info(f"Successfully deleted file from S3: {s3_key}")

        # Delete template.json from S3
        s3_key = f"{check_document[0]['transaction_id']}/template/template.json"
        s3_client.delete_object(Bucket=S3_BUCKET_NAME, Key=s3_key)
        logger.info(f"Successfully deleted template.json from S3: {s3_key}")

        await execute_query(
            query=delete_document_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        return JSONResponse(
            content={},
            status_code=204,
        )

    except Exception as e:
        logger.error(f"Failed to delete document id {id}: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to delete document id {id}"},
            status_code=500,
        )

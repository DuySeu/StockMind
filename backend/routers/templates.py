import uuid
import asyncpg
import fastapi
import json

from pydantic import BaseModel
from logsim import CustomLogger
from fastapi import APIRouter
from fastapi.responses import JSONResponse, Response

from dependencies.main import get_db, get_request_id
from core.database import execute_query

# Setup logger
logger = CustomLogger()

# Setup router
router = APIRouter(prefix="/templates", tags=["templates"])


# Model for Request Templates
class Templates(BaseModel):
    name: str
    description: str | None = None
    field: dict[str, str]
    prompt: str
    rule_ids: list[str] = []


@router.get("")
async def get_templates(
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Get all templates or a limited number"""

    query = """
        SELECT t.id, t.name, t.description, t.field, t.prompt, t.created_at, t.updated_at, COALESCE(
            json_agg(trm.rule_id) FILTER (WHERE trm.rule_id IS NOT NULL),
            '[]'::json
        ) as rule_ids
        FROM templates t
        LEFT JOIN template_rule_mapping trm ON t.id = trm.template_id
        GROUP BY t.id, t.name, t.description, t.field, t.prompt, t.created_at, t.updated_at
        ORDER BY t.created_at DESC
    """
    try:
        templates = await execute_query(
            query=query,
            params=(),
            request_id=request_id,
            connection=conn,
        )

        for template in templates:
            if template.get("field") and isinstance(template["field"], str):
                try:
                    template["field"] = json.loads(template["field"])
                    template["rule_ids"] = json.loads(template["rule_ids"])
                except json.JSONDecodeError:
                    # If parsing fails, keep the original string
                    logger.warning(
                        f"Failed to parse field for template {template.get('id')}: {template['field']}"
                    )

        return JSONResponse(content=templates, status_code=200)
    except Exception as e:
        return JSONResponse(
            content={"message": f"Error in getting all templates: {str(e)}"},
            status_code=500,
        )


@router.get("/{id}")
async def get_template_by_id(
    id: str,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Get a specific template by ID"""
    get_template_query = """
        SELECT t.id, t.name, t.description, t.field, t.prompt, t.created_at, t.updated_at, COALESCE(
            json_agg(trm.rule_id) FILTER (WHERE trm.rule_id IS NOT NULL),
            '[]'::json
        ) as rule_ids
        FROM templates t
        LEFT JOIN template_rule_mapping trm ON t.id = trm.template_id
        WHERE t.id = $1
        GROUP BY t.id, t.name, t.description, t.field, t.prompt, t.created_at, t.updated_at
    """
    try:
        if not id:
            logger.error("The template id is empty.")
            return JSONResponse(
                content={"message": f"Failed to retrieve template with id {id}."},
                status_code=400,
            )

        template = await execute_query(
            query=get_template_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not template:
            logger.error(f"The template id {id} does not exist")
            return JSONResponse(
                content={"message": f"Template with ID {id} does not exist"},
                status_code=404,
            )

        return JSONResponse(content=template, status_code=200)

    except Exception as e:
        logger.error(f"Failed to retrieve template with id {id}: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to retrieve template with ID {id}"},
            status_code=500,
        )


# Updated API to work with list-based TEMPLATES structure
@router.post("", status_code=201)
async def create_template(
    request: Templates,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """API to create new templates"""
    # Create template id
    id = str(uuid.uuid4())

    create_template_query = """
        INSERT INTO templates (id, name, description, field, prompt)
        VALUES ($1, $2, $3, $4, $5)
    """

    create_template_rule_mapping_query = """
        INSERT INTO template_rule_mapping (template_id, rule_id)
        VALUES ($1, $2)
    """

    check_existing_template_name_query = """
        SELECT id
        FROM templates
        WHERE name = $1
    """

    try:
        name = request.name
        description = request.description
        field = request.field
        prompt = request.prompt
        rule_ids = request.rule_ids

        if not name or not field or not prompt:
            logger.error("The template must include the name.")
            return JSONResponse(
                content={"message": "The template must have name, field, and prompt"},
                status_code=400,
            )

        # Check if template name already exists in database
        existing_template = await execute_query(
            query=check_existing_template_name_query,
            params=(name,),
            request_id=request_id,
            connection=conn,
        )

        if existing_template:
            logger.error(f"Template with name {name} already exists")
            return JSONResponse(
                content={"message": f"Template with name {name} already exists"},
                status_code=400,
            )

        await execute_query(
            query=create_template_query,
            params=(
                id,
                name,
                description,
                json.dumps(field),
                prompt,
            ),
            request_id=request_id,
            connection=conn,
        )

        if len(rule_ids) > 0:
            for rule_id in rule_ids:
                await execute_query(
                    query=create_template_rule_mapping_query,
                    params=(id, rule_id),
                    request_id=request_id,
                    connection=conn,
                )

        return JSONResponse(
            content={"message": f"Successfully created template {name}"},
            status_code=201,
        )

    except Exception as e:
        logger.error(f"Failed to create {request.name}: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to create template {request.name}: {str(e)}"},
            status_code=500,
        )


@router.put("/{id}", status_code=204)
async def update_template(
    id: str,
    request: Templates,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Edit a template by ID"""

    update_template_query = """
        UPDATE templates
        SET name = $1, description = $2, field = $3, prompt = $4
        WHERE id = $5
    """

    create_template_rule_mapping_query = """
        INSERT INTO template_rule_mapping (template_id, rule_id)
        VALUES ($1, $2)
    """

    check_existing_template_name_query = """
        SELECT name
        FROM templates
        WHERE name = $1 AND id != $2
    """

    check_existing_template_id_query = """
        SELECT id
        FROM templates
        WHERE id = $1
    """

    check_existing_rule_ids_query = """
        SELECT template_id, rule_id
        FROM template_rule_mapping
        WHERE template_id = $1 AND rule_id = $2
    """

    try:
        if not id:
            logger.error("Error in editing template: template id must not be empty")
            return JSONResponse(
                content={
                    "message": "Failed to edit template. The template ID is empty."
                },
                status_code=400,
            )

        existing_template = await execute_query(
            query=check_existing_template_id_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not existing_template:
            return JSONResponse(
                content={"message": f"Template with id {id} not found"},
                status_code=404,
            )

        name = request.name
        description = request.description
        field = request.field
        prompt = request.prompt
        rule_ids = request.rule_ids

        if not name or not field or not prompt:
            logger.error("The template must include the name.")
            return JSONResponse(
                content={"message": "The template must have name, field, and prompt"},
                status_code=400,
            )

        # Check if template exists in database
        existing_template = await execute_query(
            query=check_existing_template_name_query,
            params=(name, id),
            request_id=request_id,
            connection=conn,
        )

        if existing_template:
            logger.error(f"Template with name {name} already exists")
            return JSONResponse(
                content={"message": f"Template with name {name} already exists"},
                status_code=400,
            )

        if len(rule_ids) > 0:
            for rule_id in rule_ids:
                # Check if the mapping already exists
                existing_mapping = await execute_query(
                    query=check_existing_rule_ids_query,
                    params=(id, rule_id),
                    request_id=request_id,
                    connection=conn,
                )

                # Only create mapping if it doesn't already exist
                if not existing_mapping:
                    await execute_query(
                        query=create_template_rule_mapping_query,
                        params=(id, rule_id),
                        request_id=request_id,
                        connection=conn,
                    )

        await execute_query(
            query=update_template_query,
            params=(name, description, json.dumps(field), prompt, id),
            request_id=request_id,
            connection=conn,
        )

        return Response(status_code=204)

    except Exception as e:
        logger.exception(
            f"Unexpected error occurred while editing template {id}: {str(e)}"
        )
        return JSONResponse(
            content={"message": f"An unexpected error occurred: {str(e)}"},
            status_code=500,
        )


@router.delete("/{id}", status_code=204)
async def delete_template(
    id: str,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Delete a template by ID"""

    check_existing_template_name_query = """
        SELECT id
        FROM templates
        WHERE id = $1
    """

    delete_template_query = """
        DELETE FROM templates
        WHERE id = $1
    """

    delete_template_rule_mapping_query = """
        DELETE FROM template_rule_mapping
        WHERE template_id = $1
    """

    try:
        # Check if template exists in database
        existing_template = await execute_query(
            query=check_existing_template_name_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not existing_template:
            return JSONResponse(
                content={"message": f"Template with id {id} not found"},
                status_code=404,
            )

        await execute_query(
            query=delete_template_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        await execute_query(
            query=delete_template_rule_mapping_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        return Response(status_code=204)

    except Exception as e:
        logger.error(f"Failed to delete template id {id}: {str(e)}")
        return JSONResponse(
            content={"message": f"Failed to delete template id {id}"},
            status_code=500,
        )

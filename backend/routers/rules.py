import uuid
import asyncpg
import fastapi
import os
import json

from logsim import CustomLogger
from fastapi import APIRouter
from fastapi.responses import JSONResponse, Response
from pydantic import BaseModel

from dependencies.main import get_db, get_request_id
from core.database import execute_query

# Setup router for rules
router = APIRouter(prefix="/rules", tags=["rules"])
logger = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


# Model for Request Rules
class Rules(BaseModel):
    name: str
    description: str | None = None
    rule_type: str
    condition: list[dict[str, str]]
    action: dict[str, str]


@router.get("")
async def get_rules(
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Get all available rules in the system"""
    # Get all rules from database
    query = """
        SELECT id, name, description, rule_type, condition, action, created_at, updated_at
        FROM rules
        ORDER BY created_at DESC
    """
    try:
        rules = await execute_query(
            query=query,
            params=(),
            request_id=request_id,
            connection=conn,
        )

        for rule in rules:
            if (
                rule.get("condition")
                and isinstance(rule["condition"], str)
                and rule.get("action")
                and isinstance(rule["action"], str)
            ):
                try:
                    rule["condition"] = json.loads(rule["condition"])
                    rule["action"] = json.loads(rule["action"])
                except json.JSONDecodeError:
                    # If parsing fails, keep the original string
                    logger.warning(
                        f"Failed to parse condition for rule {rule.get('id')}: {rule['condition']}"
                    )

        return JSONResponse(
            content=rules,
            status_code=200,
        )
    except Exception as e:
        logger.exception(f"Error getting rules: {e}")
        return JSONResponse(
            content={"message": "Error getting rules"},
            status_code=500,
        )


@router.post(path="", status_code=201)
async def create_rule(
    request: Rules,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Create a new custom rule"""

    # Create rule id
    id = str(uuid.uuid4())

    create_rule_query = """
        INSERT INTO rules (id, name, description, rule_type, condition, action)
        VALUES ($1, $2, $3, $4, $5, $6)
    """

    check_existing_rule_query = """
        SELECT id
        FROM rules
        WHERE name = $1
    """

    try:
        # Validate required fields
        name = request.name
        description = request.description
        rule_type = request.rule_type
        condition = request.condition
        action = request.action

        if not name or not condition or not action or not rule_type:
            logger.error("The rule must include the name and condition and action")
            return JSONResponse(
                content={"message": "The rule must have name and condition and action"},
                status_code=400,
            )

        # Check if rule name already exists in database
        existing_rule = await execute_query(
            query=check_existing_rule_query,
            params=(name,),
            request_id=request_id,
            connection=conn,
        )
        if existing_rule:
            logger.error(f"Rule with name {name} already exists")
            return JSONResponse(
                content={"message": f"Rule with name {name} already exists"},
                status_code=400,
            )

        # Insert new role into database
        params = (
            id,
            name,
            description,
            rule_type,
            json.dumps(condition),
            json.dumps(action),
        )
        await execute_query(
            query=create_rule_query,
            params=params,
            request_id=request_id,
            connection=conn,
        )

        return JSONResponse(
            content={"message": f"Successfully created rule {name}"},
            status_code=201,
        )
    except Exception as e:
        logger.exception(f"Error creating rule: {e}")
        return JSONResponse(
            content={"message": f"Error creating rule {name}: {e}"},
            status_code=500,
        )


@router.put(path="/{id}", status_code=204)
async def update_rule(
    id: str,
    request: Rules,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Update an existing rule"""

    update_rule_query = """
        UPDATE rules
        SET name = $1, description = $2, rule_type = $3, condition = $4, action = $5
        WHERE id = $6
    """

    check_existing_rule_name_query = """
        SELECT id
        FROM rules
        WHERE name = $1 AND id != $2
    """

    check_existing_rule_id_query = """
        SELECT id
        FROM rules
        WHERE id = $1
    """

    # Read the JSON body
    try:
        if not id:
            logger.error("Error in editing rule: rule id must not be empty")
            return JSONResponse(
                content={"message": "Failed to edit rule. The rule ID is empty."},
                status_code=400,
            )

        # Check if rule exists in database
        existing_rule = await execute_query(
            query=check_existing_rule_id_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not existing_rule:
            return JSONResponse(
                content={"message": f"Rule with id {id} not found"},
                status_code=404,
            )

        name = request.name
        description = request.description
        rule_type = request.rule_type
        condition = request.condition
        action = request.action

        if not name or not rule_type or not condition or not action:
            logger.error("The rule must include the name and condition and action")
            return JSONResponse(
                content={"message": "The rule must have name and condition and action"},
                status_code=400,
            )

        # Check if rule name already exists in database
        check_existing_rule_name = await execute_query(
            query=check_existing_rule_name_query,
            params=(name, id),
            request_id=request_id,
            connection=conn,
        )

        if check_existing_rule_name:
            logger.error(f"Rule with name {name} already exists")
            return JSONResponse(
                content={"message": f"Rule with name {name} already exists"},
                status_code=400,
            )

        await execute_query(
            query=update_rule_query,
            params=(
                name,
                description,
                rule_type,
                json.dumps(condition),
                json.dumps(action),
                id,
            ),
            request_id=request_id,
            connection=conn,
        )

        return Response(status_code=204)
    except Exception as e:
        logger.exception(f"Error updating rule: {e}")
        return JSONResponse(
            content={"message": f"Error updating rule {name}: {e}"},
            status_code=400,
        )


@router.delete(path="/{id}", status_code=204)
async def delete_rule(
    id: str,
    conn: asyncpg.Connection = fastapi.Depends(get_db),
    request_id: str = fastapi.Depends(get_request_id),
):
    """Delete a custom rule and all related mappings"""

    # First check if rule exists
    check_rule_query = """
        SELECT id, name FROM rules WHERE id = $1
    """

    delete_mapping_query = """
        DELETE FROM template_rule_mapping WHERE rule_id = $1
    """

    delete_rule_query = """
        DELETE FROM rules WHERE id = $1
    """

    try:
        # Check if rule exists
        existing_rule = await execute_query(
            query=check_rule_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        if not existing_rule:
            return JSONResponse(
                content={"message": f"Rule with id {id} not found"},
                status_code=404,
            )

        # Then delete the rule in rules table
        await execute_query(
            query=delete_rule_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        # Delete related mappings in template_rule_mapping table
        await execute_query(
            query=delete_mapping_query,
            params=(id,),
            request_id=request_id,
            connection=conn,
        )

        return Response(status_code=204)
    except Exception as e:
        logger.exception(f"Error deleting rule: {e}")
        return JSONResponse(
            content={"message": f"Error deleting rule {id}: {e}"},
            status_code=400,
        )

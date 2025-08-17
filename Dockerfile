FROM 271540607717.dkr.ecr.ap-southeast-1.amazonaws.com/secure-vl-base/nodejs:20 AS fe_build

COPY --chown=apps:apps web /home/apps

WORKDIR /home/apps

RUN npm install && \
    npm run build

FROM 271540607717.dkr.ecr.ap-southeast-1.amazonaws.com/secure-vl-base/python:3.12

USER apps
# Set working directory
WORKDIR /home/apps

# Copy requirements and install Python dependencies
COPY backend/requirements.txt .
RUN uv venv deploy && \
    source deploy/bin/activate && \
    uv pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY backend .
# Copy schema files to the container
COPY --chown=apps:apps schema /home/apps/schema
COPY --from=fe_build --chown=apps:apps --chmod=755 /home/apps/dist /home/apps/frontend
COPY --chown=apps:apps --chmod=755 entrypoint.sh /home/apps/entrypoint.sh

ENTRYPOINT [ "/home/apps/entrypoint.sh" ]
# Command to run the application
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
TAG = 1.0.0
REGION = ap-southeast-1
ACCOUNT_ID = 271540607717
ACCOUNT_ID = 302010997939
ECR_REGISTRY = $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com

clean:
	docker-compose down --volumes

build:
	docker-compose build

run: build
	docker-compose up -d

down:
	docker-compose down

restart: down run

login:
	aws ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin 271540607717.dkr.ecr.$(REGION).amazonaws.com
	aws ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin 302010997939.dkr.ecr.$(REGION).amazonaws.com
	aws ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin $(ECR_REGISTRY)

logout:
	docker logout $(ECR_REGISTRY)
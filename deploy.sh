#!/bin/bash

# 部署脚本 - 在 VPS 上运行

set -e

echo "======================================"
echo "Blog Backend Deployment Script"
echo "======================================"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 .env 文件
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found!${NC}"
    echo "Please create .env file with required variables:"
    echo "  DB_USER=blog"
    echo "  DB_PASSWORD=your-strong-password"
    echo "  DB_NAME=blog"
    echo "  JWT_SECRET=your-secret-key"
    exit 1
fi

# 拉取最新镜像
echo -e "${YELLOW}Pulling latest Docker image...${NC}"
docker pull ghcr.io/your-username/blog-backend:latest

# 停止旧容器
echo -e "${YELLOW}Stopping old containers...${NC}"
docker-compose -f docker-compose.prod.yml down

# 启动新容器
echo -e "${YELLOW}Starting new containers...${NC}"
docker-compose -f docker-compose.prod.yml up -d

# 等待服务启动
echo -e "${YELLOW}Waiting for services to be healthy...${NC}"
sleep 10

# 检查服务状态
if docker-compose -f docker-compose.prod.yml ps | grep -q "Up"; then
    echo -e "${GREEN}✓ Deployment successful!${NC}"
    docker-compose -f docker-compose.prod.yml ps
else
    echo -e "${RED}✗ Deployment failed!${NC}"
    echo "Logs:"
    docker-compose -f docker-compose.prod.yml logs --tail=50
    exit 1
fi

# 清理旧镜像
echo -e "${YELLOW}Cleaning up old images...${NC}"
docker image prune -f

echo -e "${GREEN}======================================"
echo "Deployment complete!"
echo "======================================${NC}"

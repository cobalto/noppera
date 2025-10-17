#!/bin/bash

echo "Setting up Noppera Image Board API for local development..."

# Check if PostgreSQL is installed
if ! command -v psql &> /dev/null; then
    echo "PostgreSQL is not installed. Please install PostgreSQL first."
    echo "On macOS: brew install postgresql"
    echo "On Ubuntu: sudo apt-get install postgresql postgresql-contrib"
    exit 1
fi

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 &> /dev/null; then
    echo "PostgreSQL is not running. Please start PostgreSQL service."
    echo "On macOS: brew services start postgresql"
    echo "On Ubuntu: sudo systemctl start postgresql"
    exit 1
fi

# Create database and user
echo "Creating database and user..."
sudo -u postgres psql -c "CREATE USER admin WITH PASSWORD 'password';" 2>/dev/null || echo "User 'admin' already exists"
sudo -u postgres psql -c "CREATE DATABASE imageboard OWNER admin;" 2>/dev/null || echo "Database 'imageboard' already exists"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE imageboard TO admin;" 2>/dev/null

# Initialize database schema
echo "Initializing database schema..."
PGPASSWORD=password psql -h localhost -U admin -d imageboard -f init.sql

# Create uploads directory
echo "Creating uploads directory..."
mkdir -p uploads

# Copy environment file
if [ ! -f .env ]; then
    echo "Creating .env file..."
    cp .env.example .env
    echo "Please edit .env file with your configuration if needed."
fi

# Build the application
echo "Building application..."
go build -o noppera ./cmd/api

echo "Setup complete!"
echo ""
echo "To run the application:"
echo "  ./noppera"
echo ""
echo "Or with environment variables:"
echo "  source .env && ./noppera"
echo ""
echo "API will be available at: http://localhost:8080"
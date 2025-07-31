.PHONY: help install install-all clean lint format type-check test run build all check

# Default target
help:
	@echo "Available commands:"
	@echo "  install     - Install production dependencies only"
	@echo "  install-all - Install all dependencies (production + dev + extras)"
	@echo "  clean       - Clean up build artifacts and cache"
	@echo "  lint        - Run ruff linter"
	@echo "  format      - Format code with ruff and isort"
	@echo "  type-check  - Run pyright type checking"
	@echo "  test        - Run pytest tests"
	@echo "  run         - Run the application"
	@echo "  build       - Build the package"
	@echo "  check       - Run all code quality checks (lint + type-check)"
	@echo "  all         - Run format, check, and test"

# Installation targets
install:
	uv sync --no-dev

install-all:
	uv sync --all-groups --all-extras

# Clean up
clean:
	rm -rf dist/
	rm -rf .uv_cache/
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} +

# Code quality
lint:
	uv run ruff check src/

format:
	uv run ruff format src/
	uv run isort src/

type-check:
	uv run pyright src/

# Testing
test:
	uv run pytest tests/ -v

# Development
run:
	uv run cyro

# Build
build:
	uv build

# Combined targets
check: lint type-check

all: format check test

# Fix common issues
fix:
	uv run ruff check --fix src/
	uv run ruff format src/
	uv run isort src/
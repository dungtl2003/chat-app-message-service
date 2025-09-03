# Message service

Last updated: 2025-09-03

# Table of Contents    

- [Description](#description)
- [Endpoints](#endpoints)
    - [GET /healthcheck](#get-healthcheck)
- [Configuration](#configuration)
- [Running](#running)
- [Testing](#testing)

## Description

This service is responsible for managing media files, including uploading, processing, and serving media content. It provides an API for interacting with media files and integrates with other services for storage and processing.

## Endpoints

### GET /healthcheck

- **Description**: Check the health status of the service.
- **Response**: JSON object with the health status.

## Configuration

| Name | Description | Required | Default | Type | Possible Values  |
| ---- | ----------- | -------- | ------- | ---- | ---------------- |
| DATABASE_URL | The URL to the database service | Yes | | string | any valid URL |
| PORT | The port to run the service | No | 8400 | int | any valid port number |
| LOG_LEVEL | The log level of the service | No | INFO | string | DEBUG, INFO, WARN, ERROR |
| LOG_KIND | The kind of log to output | No | TEXT | string | TEXT, JSON |
| ID_GENERATOR_ADDR | The address of the id generator service | Yes | | string | any valid string made of address and port (e.g. localhost:8501) |
| ID_GENERATOR_CERT_DIR | The directory to the certificate of the id generator service | No | | string | any valid directory |
| ENVIRONMENT | The environment the service is running in | No | dev | string | dev, prod, test |
| MEDIA_SERVICE_URL | The URL to the media service | Yes | | string | any valid URL |

You can see the full configuration example in `./environments/.env.template` file.

## Running

Run the service in development mode with the following command:

```bash
make run_with_services
```

This will spin up the dependency services and run the program in development mode.

## Testing

Run the tests with the following command:

```bash
make test
```

Or run the tests with analysis enabled:

```bash
make test JSON=1
```

There are scripts that support logging services and result to file. If in the future, you want to add a new service log, you need to update `LOG_META` variable in `./scripts/test_local.sh` script.

## Docker

Use this command to build the Docker image:

```bash
make ci_%env # e.g. make ci_dev
```

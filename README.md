# heist

A work-in-progress to learn more about Discord bots and MongoDB, using the Go programming language.

## Setting The Environment Variables

Within the `.env` file, you should configure the following values. This `.env` file is passed to
both the `heist_bot` containier as well as the `heist_mongo` container.

``` bash
# Discord Bot Configuration
BOT_TOKEN="<bot_token>"
APP_ID="<app_id>"

# Heist State Storage Method. Choose one or the other.
# HEIST_STORE="file"
HEIST_STORE="mongodb"

# Heist Theme Configruation. They must be stored in "./config/heist/".
HEIST_DEFAULT_THEME="<default_theme>"
HEIST_THEME_DIR="./configs/heist/"

# MongoDB admin configuration. For MONGODB_ADMIN_DB, this has been tested with a value of "admin".
MONGODB_ADMIN_DB="<admin_db>"
MONGO_ROOT_USERNAME="<root_username>"
MONGO_ROOT_USERNAME="<root_password>"

# Heist DB configuration. These are only needed if your HEIST_STORE is set to "mongodb". You should log into
# your MongoDB and create the database and configure it so the userid/password credentials grant read/write access.
MONGODB_URI="mongodb://heist_mongo:27017/?connect=direct"
MONGODB_USERID="<heist_db_userid>"
MONGODB_PASSWORD="<heist_db_pasword>"
MONGODB_DATABASE="<heist_db>"
```

## Running the Heist Bot

### Run as a Standalone Application

When developing, you can use

```bash
go run cmd/heist/main.go
```

to compile and run the heist bot. Once it is stable, you can use the `make` command to generate a binary that you can
execute.

### Run as a Docker Image

#### Build Container

You should edit your `.env` file to set `HEIST_STORE="mongodb"` when deploying in this manner. You should manually start
a MongoDB instance that your deployed image may use for access.

``` bash
docker-buildx build -t heist:1.0.0 .
docker push heist:1.0.0
```
#### Start Container

```bash
docker run --env BOT_TOKEN="<bot-token>" --env APP_ID="<app-id>" --env HEIST_DEFAULT_THEME="<theme-name>" --env HEIST_THEME_DIR="<theme-dir>" --name <container-name> heist:1.0.0
```

### Run using `docker-compose`

The following command will both build the container, as well as deploy with both the heist bot as well as MongoDB. You should edit your
`.env` file to set `HEIST_STORE="mongodb"` when deploying in this manner.

```bash
docker-compose up --build
```
# heist

A work-in-progress to learn more about Discord bots and MongoDB, using the Go programming language.

## Running the Heist Bot

### Configuring the Heist Bot

The heist bot relies on a set of environment variables to configure it.

#### Heist Bot

```bash
# Discord Bot Configuration
BOT_TOKEN="<bot_token>"
APP_ID="<bot_application_id>"

# Heist State Storage Method. Choose one or the other. If you are deploying to a
# container, then you should be using "mongodb", otherwise the data will not be
# persisted.
# HEIST_STORE="file"
HEIST_STORE="mongodb"

# Heist File Store Configuration. If running within a container, you must use the
# values below. If running as a stand-alone application, you can change them to your
# preferred location.
HEIST_FILE_STORE_DIR="./store/heist/"
HEIST_FILE_NAME="heist.json"

# You can use this variable to point at a development server, in which case any
# changes you have made will only appear on the development server.
# HEIST_GUILD_ID="<server ID>"

# Heist Theme Configruation.  If running within a container, you must use the values
# below. If running as a stand-alone application, you can change them to your
# preferred location.
HEIST_DEFAULT_THEME="clash"
HEIST_FILE_THEME_DIR="./store/theme/"

# Heist DB configuration. This example shows you how to connect to MongoDB within a
# container, where the name of the deployed MongoDB container is `heist_mongo`. If 
# running outside a container, replace `heist_mongo` with the IP address or DNS name
# of your MongoDB instance. Prior to Heist being able to connect to the MongoDB
# instance, create the MongoDB database and add a database user for the database
# with Read/Write permissions, and use those values below.
MONGODB_URI="mongodb://heist_mongo:27017/?connect=direct"
MONGODB_USERID="<heist_userid>"
MONGODB_PASSWORD="<heist_password<"
MONGODB_DATABASE="<heist_db_name>"
```

#### MongoDB

If you are deploying MongoDB via `docker compose`, then the following values should
be configured for the MongoDB database. Unless you have a unique admin database,
`MONGODB_ADMIN_DB` should be set to `admin`.

```bash
# MongoDB admin configuration. You should configure the administration database
# (defaults to `admin`), along with the username and password that should be used
# when deploying the MongoDB database.
MONGODB_ADMIN_DB="<admin_database>"
MONGO_INITDB_ROOT_USERNAME="<root_username>"
MONGO_INITDB_ROOT_USERNAME="<root_password>"
```

#### Configuring MongoDB for the Heist Bot

The MongoDB database needs to have a user configured who can read and write from the heist database. Using the
`mongosh` command to connect to your MongoDB instance, including any username/password credentials that may be
required, you can add a user by using the following command. For example, you may need to specify a command
such as this if the MongoDB instance is running locally:

```bash
mongosh -host 'localhost:27017' -u <root_username> -p <root_password>
```

Once mongosh has started, enter the following command to create a user who can read and write to the specified
database.

```bash
use admin
db.createUser(
  {
    user: "<heist_db_userid>",
    pwd: "<heist_db_pwd>",
    roles: [ { role: "readWrite", db: "<heist_db_name>" } ]
  }
)
```

Note that the actual MongoDB database will be created when the first collection or document is written to the database.

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
docker run --envfile ./.env --name <container-name> heist:1.0.0
```

### Run using `docker compose`

The following command will both build the container, as well as deploy with both the heist bot as well as MongoDB. You should edit your
`.env` file to set `HEIST_STORE="mongodb"` when deploying in this manner.

```bash
docker compose up --build
```

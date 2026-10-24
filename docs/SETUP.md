# Setup and storage

Use the root README for development commands. Use build.sh for an executable with an embedded interface. Compose builds the application and starts PostgreSQL.

The API defaults to port 8787, the recording proxy to 8788 and the development frontend to 3000. Bind local installations to loopback or configure an appropriate remote access boundary.

## SQLite

Set BLACKSAND_DB_DRIVER=sqlite. The default file is data/blacksand.db under the application base directory. BLACKSAND_SQLITE_PATH selects another file.

A config.json alternative is:

~~~json
{"database":{"driver":"sqlite","path":"data/blacksand.db"}}
~~~

Explicit relative SQLite paths resolve from the working directory. Keep the launch directory consistent or use an absolute path.

## PostgreSQL

Copy config.example.json to config.json and fill in the connection, or set BLACKSAND_PG_DSN. An example is postgres://blacksand:YOUR_PASSWORD@127.0.0.1:5432/blacksand?sslmode=require. Select TLS settings suitable for your server. Compose configures its own internal database service.

## Configuration and first launch

BLACKSAND_CONFIG selects a configuration file. Otherwise the application searches the working and executable directories. Database environment settings can override file settings. Agent, model and integration settings are stored in the domain database.

The -data flag selects the working data directory; it does not override an explicit database connection or SQLite path. BLACKSAND_SKILL_DIR selects the skill root.

Complete /setup, then sign in as blacksand and configure a model. Install tool dependencies required by your selected integrations.

## Existing installations and backups

Changing storage settings is not an automatic migration. Back up the database and associated evidence/workspace files together. Use a database-aware SQLite backup or stop writers before copying its files. Protect credentials and jwt.key. Test upgrades against a backup before replacing the live installation.

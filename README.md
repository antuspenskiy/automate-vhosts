# Automate virtual hosts

## Tasks

### Dump MySQL database

- Dump database from `server X` via `mysqldump` and `gzip` it in local disk then rsync it to remote storage.
- Store dumps for X days and rotate it.

### Import MySQL database

- Check if remote storage exists.
- `Rsync` database dump from remote storage to local disk on `server Y`. `Gunzip` it and import dump via `mysql`.
- Use `config.json` for environment variables.
- Clear copied dump and extracted files when import succefull.

### Gitlab Schedules Pipeline

- Setting Gitlab Schedules for compiled file and CI to run them.

## Database dump (dbdump)

```bash
Usage of ./dbdump:
  -backup-dir string
    Backup directory for dumps. (default "/opt/backup/db")
  -db string
    Database name. (default "test")
  -db-all
    If set dump all Mysql databases.
  -gzip
    If set gzip compression enabled. (default true)
  -h string
    Name of your Mysql hostname. (default "localhost")
  -storage-dir string
    Remote storage directory for dumps. (default "/mnt/backup")
  -u string
    Name of your database user. (default "test")
```

## Import database dump (dbimport)

```bash
{
  "Test": {
    "test": "",
    "hostname": "",
  },
  "Prod": {
    "production": "",
    "hostname": "",
  },
  "rootDir": "/var/web",
  "dbDir": "/opt/backup/db",
  "storageDir": "/mnt/backup"
}

Usage of ./dbimport:
  -branch string
    Branch name (default "1-test-branch")
  -database string
    Name of your database.
  -hostname string
    Name of your database hostname. (default "localhost")
  -password string
    Name of your database user password.
  -port string
    Name of your database port. (default "3306")
  -user string
    Name of your database user.
```

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

### Prepare virtual hosts

- Build project for virtual host
- Parse settings for virtual host using Gitlab variables

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

## Prepare virtual hosts (prepare)

```bash
Usage of ./prepare:
  -CI_COMMIT_REF_NAME string
    	The branch or tag name for which project is built.
  -CI_COMMIT_REF_SLUG string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
  -CI_COMMIT_SHA string
    	The commit revision for which project is built.
```

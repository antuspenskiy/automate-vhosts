# Automate virtual hosts

### Dump MySQL database

- Dump database from `server X` via `mysqldump` and `gzip` it in local disk then rsync it to remote storage.
- Store dumps for X days and rotate it.

### Import MySQL database

- `rsync` database dump from remote storage to local disk on `server Y`. `gunzip` it and import dump via `mysql`.
- Extract archived dump.
- Connect to MySQL create database and user, grant privileges.
- Import dump via `mysql`.
- Delete dump files.

### Prepare virtual hosts

- Build project for virtual host.
- Parse settings for virtual host using Gitlab variables.
- Create env.json for library module.

### Create configuration files for virtual hosts

- Create nginx,php-fpm and pm2 configuration files.
- Restart nginx, php-fpm when create configuration files.
- Start pm2 process from configuration file.
- Reload pm2 process if .json file exists, run commands for checkout and build if virtual host directory exists.

### Delete configuration files

- Delete virtual host directory.
- Delete virtual host nginx,php-fpm and pm2 configuration files.
- Drop MySQL database of virtual host if exists. 

### Gitlab Schedules Pipeline

- Setting Gitlab Schedules for `dbdump` and CI to run them.

## Database dump (dbdump)

```bash
Version    : 1.0.0
Git Hash   : e1ecc62d052bd8b8e1b44c0161bd0b7577a9863b
Build Time : 2017-09-20_10:07:19

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
Version    : 1.0.0
Git Hash   : e1ecc62d052bd8b8e1b44c0161bd0b7577a9863b
Build Time : 2017-09-20_10:07:19

Usage of ./dbimport:
  -CI_COMMIT_REF_SLUG string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
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
Version    : 1.0.0
Git Hash   : e1ecc62d052bd8b8e1b44c0161bd0b7577a9863b
Build Time : 2017-09-20_10:07:19

Usage of ./prepare:
  -CI_COMMIT_REF_NAME string
    	The branch or tag name for which project is built.
  -CI_COMMIT_REF_SLUG string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
  -CI_COMMIT_SHA string
    	The commit revision for which project is built.
```

## Create configuration files for virtual hosts (createconfigs)

```bash
Version    : 1.0.0
Git Hash   : e1ecc62d052bd8b8e1b44c0161bd0b7577a9863b
Build Time : 2017-09-20_10:07:19

Usage of ./createconfigs:
  -CI_COMMIT_REF_SLUG string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.    	
```

## Delete configuration files for virtual hosts (deletestuff)

```bash
Version    : 1.0.0
Git Hash   : e1ecc62d052bd8b8e1b44c0161bd0b7577a9863b
Build Time : 2017-09-20_10:07:19

Usage of ./deletestuff:
  -CI_COMMIT_REF_SLUG string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
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

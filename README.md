# Automate virtual hosts

A bunch of CLI utilities for automating virtual hosts in different environments and servers via Gitlab CI. Using https://github.com/spf13/viper for reading config files and parse strings.
Examples of config files in config directory.

### Dump MySQL database

- Dump database from `server X` via `mysqldump`. Archive dumps with `gzip` and save it in local disk then rsync it to remote storage.
- Store dumps for X days and rotate it.

### Import MySQL database

> Some MySQL commands can run on different servers. 

- `rsync` database dump from remote storage to local disk on `server Y`. `gunzip` it and import dump via `mysql`.
- Extract archived dump.
- Connect to MySQL create database and user, grant privileges.
- Import database dump via `mysql`.
- Delete database dump files by extension.

### Prepare virtual hosts

> Some commands can run on different servers.

- Run commands for checkout and build if virtual host directory exists.
- Run another commands for checkout and build if virtual host directory not exists.
- Parse settings for virtual host using Gitlab variables.
- Create env.json environment configuration for Library module from template.
- Create Laravel .env.json environment configuration from template.


### Create configuration files for virtual hosts

> Some commands can run on different servers.

- Create nginx,php-fpm and pm2 configuration files from template.
- Restart nginx, php-fpm when create configuration files.
- Start pm2 process from configuration file.
- Reload pm2 process if `json` file exists.

### Delete configuration files

> Some commands can run on different servers.

- Delete virtual host directory.
- Delete virtual host nginx,php-fpm and pm2 configuration files.
- Drop MySQL database of virtual host if exists. 

### Gitlab Schedules Pipeline

- Setting Gitlab Schedules for `dbdump` and CI to run them.

### Gitlab CI Example

We can use Secret Variables from Gitlab CI and pass them as arguments in utilities. Also we can use Docker containers and connect to servers from containers via SSH using Gitlab CI Secret Variables.

```yaml
image: "your_custom_docker_images"

.default-cache: &default-cache
  key: $CI_COMMIT_REF_NAME
  paths:
    - php_packages
    - node_modules
    - ~/.composer/cache/files
    - ~/.yarn-cache

.push-cache: &push-cache
  cache:
    <<: *default-cache
    policy: push

.pull-cache: &pull-cache
  cache:
    <<: *default-cache
    policy: pull

variables:
  MARIADB_ROOT_PASSWORD: root
  MARIADB_USER: homestead
  MARIADB_PASSWORD: secret
  MARIADB_DATABASE: homestead
  DB_HOST: mariadb

stages:
  - lint
  - import
  - host
  - review
  - deploy

.dedicated-runner: &dedicated-runner
  tags:
    - your_gitlab_runner_tag

.use-mariadb: &use-mariadb
  services:
    - webhippie/mariadb:latest

prepare_lint:
  <<: *use-mariadb
  <<: *dedicated-runner
  stage: lint
  cache:
    <<: *default-cache
  script:
     - your_lint_commands

prepare_import:
  <<: *dedicated-runner
  stage: import
  image: alpine:latest
  before_script:
    - 'which ssh-agent || (apk --update add openssh bash)'
    - eval $(ssh-agent -s)
    - bash -c "ssh-add <(echo '$SSH_PRIVATE_KEY_TEST')"
    - mkdir -p ~/.ssh
    - ssh-keyscan -H 'your_server_hostname' >> ~/.ssh/known_hosts
    - ssh-keyscan your_server_hostname | sort -u - ~/.ssh/known_hosts -o ~/.ssh/known_hosts
    - '[[ -f /.dockerenv ]] && echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config'
  script:
    - import
  variables:
    GIT_STRATEGY: none
  only:
    - /.*-test/

prepare_host:
  <<: *dedicated-runner
  stage: host
  image: alpine:latest
  before_script:
    - 'which ssh-agent || (apk --update add openssh bash)'
    - eval $(ssh-agent -s)
    - bash -c "ssh-add <(echo '$SSH_PRIVATE_KEY_TEST')"
    - mkdir -p ~/.ssh
    - ssh-keyscan -H 'your_server_hostname' >> ~/.ssh/known_hosts
    - ssh-keyscan your_server_hostname | sort -u - ~/.ssh/known_hosts -o ~/.ssh/known_hosts
    - '[[ -f /.dockerenv ]] && echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config'
  script:
    - ssh $SERVER_TEST /path/to/prepare -refslug=$CI_COMMIT_REF_SLUG -commitsha=$CI_COMMIT_SHA
  variables:
    GIT_STRATEGY: none
  only:
    - /.*-test/

review:
  <<: *dedicated-runner
  stage: review
  image: alpine:latest
  before_script:
    - 'which ssh-agent || (apk --update add openssh bash)'
    - eval $(ssh-agent -s)
    - bash -c "ssh-add <(echo '$SSH_PRIVATE_KEY_TEST')"
    - mkdir -p ~/.ssh
    - ssh-keyscan -H 'your_server_hostname' >> ~/.ssh/known_hosts
    - ssh-keyscan your_server_hostname | sort -u - ~/.ssh/known_hosts -o ~/.ssh/known_hosts
    - '[[ -f /.dockerenv ]] && echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config'
  script:
    - ssh -T $SERVER_TEST sudo /path/to/createconfigs -refslug=$CI_COMMIT_REF_SLUG
  variables:
    GIT_STRATEGY: none
  environment:
    name: review/$CI_BUILD_REF_NAME
    url: https://$CI_BUILD_REF_SLUG.$APPS_DOMAIN
  only:
    - /.*-test/

stop_review:
  <<: *dedicated-runner
  stage: review
  image: alpine:latest
  before_script:
    - 'which ssh-agent || (apk --update add openssh bash)'
    - eval $(ssh-agent -s)
    - bash -c "ssh-add <(echo '$SSH_PRIVATE_KEY_TEST')"
    - mkdir -p ~/.ssh
    - ssh-keyscan -H 'your_server_hostname' >> ~/.ssh/known_hosts
    - ssh-keyscan your_server_hostname | sort -u - ~/.ssh/known_hosts -o ~/.ssh/known_hosts
    - '[[ -f /.dockerenv ]] && echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config'
  script:
    - av-remove
  variables:
    GIT_STRATEGY: none
  when: manual
  environment:
    name: review/$CI_BUILD_REF_NAME
    action: stop
  only:
    - /.*-test/

deploy:
  <<: *dedicated-runner
  stage: deploy
  image: alpine:latest
  before_script:
    - 'which ssh-agent || (apk --update add openssh bash)'
    - eval $(ssh-agent -s)
    - bash -c "ssh-add <(echo '$SSH_PRIVATE_KEY')"
    - mkdir -p ~/.ssh
    - ssh-keyscan -H 'your_server_hostname' >> ~/.ssh/known_hosts
    - ssh-keyscan your_server_hostname | sort -u - ~/.ssh/known_hosts -o ~/.ssh/known_hosts
    - '[[ -f /.dockerenv ]] && echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config'
  environment:
    name: production
    url: https://your_server
  script:
    - ssh $SERVER ./deploy.sh $CI_COMMIT_SHA
  only:
    - master
  when: on_success
```

## Database dump (dbdump)

```bash
Version    : 1.0.1-alpha
Git Hash   : 60bd0857b3c756f12b4b4b1e248c8a48ec1b49bf
Build Time : 2017-12-26_12:02:28

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
Version    : 1.0.1-alpha
Git Hash   : 60bd0857b3c756f12b4b4b1e248c8a48ec1b49bf
Build Time : 2017-12-26_12:02:28

Usage of ./dbimport:
  -database string
    	Name of your database.
  -hostname string
    	Name of your database hostname. (default "localhost")
  -password string
    	Name of your database user password.
  -port string
    	Name of your database port. (default "3306")
  -refslug string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
  -user string
    	Name of your database user.
```

## Prepare virtual hosts (prepare)

```bash
Version    : 1.0.1-alpha
Git Hash   : 60bd0857b3c756f12b4b4b1e248c8a48ec1b49bf
Build Time : 2017-12-26_12:02:28

Usage of ./prepare:
  -commitsha string
    	The commit revision for which project is built.
  -refslug string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
```

## Create configuration files for virtual hosts (createconfigs)

```bash
Version    : 1.0.1-alpha
Git Hash   : 60bd0857b3c756f12b4b4b1e248c8a48ec1b49bf
Build Time : 2017-12-26_12:02:28

Usage of ./prepare:
  -commitsha string
    	The commit revision for which project is built.
  -refslug string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
```

## Delete configuration files for virtual hosts (deletestuff)

```bash
Version    : 1.0.1-alpha
Git Hash   : 60bd0857b3c756f12b4b4b1e248c8a48ec1b49bf
Build Time : 2017-12-26_12:02:28

Usage of ./deletestuff:
  -database string
    	Name of your database.
  -hostname string
    	Name of your database hostname. (default "localhost")
  -password string
    	Name of your database user password.
  -port string
    	Name of your database port. (default "3306")
  -refslug string
    	Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.
  -user string
    	Name of your database user.
```

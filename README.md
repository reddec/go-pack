# go-pack

>> This project no more active. Use [goreleaser](https://github.com/goreleaser/goreleaser) instead

Simple golang DEBIAN packager inspired by NPM.

Features:

* build a single binary application
* build a binary `upstart` service
* package to DEB
* single configuration in one file

Checkout binary version [here](https://github.com/reddec/go-pack/releases)

### binary package

```
go-pack -c group-package_name
```
  
### upstart service

```
go-pack -c -s group-package_name
```

Now only `upstart` is supported. `systemd` is one of main task for future releases


### Build

Just run in directory with package.json

```
go-pack
```
  
or use `-d <dir_name>` for custom location

```
go-pack -d path/to/dir
```

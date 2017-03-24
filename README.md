Autopull 0.0.1
============

Autopull helps you to download and run your project
It pulls automatically if project remote repository changed and restarts project.

Install
=======

```sh
go get -u github.com/Boomerangz/autopull/
```

Commands
========

**help** – show help or help [command]

```sh
autopull -h
```

## Run your project

```sh
autopull --config {project-config}

## **config.json** file

```json
{
    "cmd": [*Your commands list*],
    "directory": "project",
    "git_repo": *Your repo*,
    "git_branch": *Your repos branch*,
    "period_in_seconds": *Period in seconds between checking project*
}

License
=======

<a rel="license" href="http://creativecommons.org/licenses/by/4.0/"><img alt="Creative Commons License" style="border-width:0" src="http://i.creativecommons.org/l/by/4.0/88x31.png" /></a><br /><span xmlns:dct="http://purl.org/dc/terms/" property="dct:title">Autopull</span> is licensed under a <a rel="license" href="http://creativecommons.org/licenses/by/4.0/">Creative Commons Attribution 4.0 International License</a>.<br />Based on a work at <a xmlns:dct="http://purl.org/dc/terms/" href="http://github.com/Boomerangz/autopull/" rel="dct:source">http://github.com/Boomerangz/autopull/</a>.
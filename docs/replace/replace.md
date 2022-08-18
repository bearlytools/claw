# Claw Replace Files

## Introduction

A Claw replace file provide for replacing a claw import with a package either in another repo or on the local file system. 

There are three ways to replace a dependency:

* `claw.mod` `replace` directive
* local.replace
* global.replace

`claw.mod` directive is covered in our section on modules. This is used to replace a bad actor's package. Here we will talk about `local.replace` and `global.replace`.

## local.replace

`local.replace` is useful for when you want to replace a dependency for development purposes. All replacements will happen for dependencies that are in the import graph when compiling from that directory.

This helps facilitate development where you want to test modifications to other dependent files in other repositories. There is no need to do this for files in the local repository.


### General syntax example of local.replace file

```local.replace
replace (
    github.com/johnsiilver/motorcycles => ../../directory/of/module
)
```

or 

```local.replace
replace (
    github.com/johnsiilver/motorcycles => github.com/newrepo/motorcycles
)
```

* `replace ()` provides the ability to replace a required module with the module on the local filesystem. This allows you to do local development on a dependent module while working on this module. THIS IS NOT REQUIRED IF THE OTHER MODULE IS IN THE SAME REPOSITORY.

Each entry must only span a single line, no commas after.

For the replace to work, the module must be imported in `claw.mod`.

The `claw replace local init` command will also supply a .gitignore file that will ignore `local.replace`. This prevents accidental checkin. If a .gitignore exists, it will be checked for `local.replace` and it if doesn't exist, it will append it.

## global.replace

A `global.replace` indicates to `clawc` that any imports of this package should instead use the replacement package.

The syntax is:

```global.replace
with github.com/newrepo/path/to/package
```
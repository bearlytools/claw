# Claw Modules

## Introduction

Claw module files provide for versioning import dependencies imported in your .claw file.

Claw modules are roughly based on the Go language modules, but are not equivalent due to the differences in supporting versioning for an IDL and support for a complete language.

Therefore, we only have a tiny subset of what Go modules provide.

`claw.mod` files are not required if all `.claw` files can be located within the same repository. If your `.claw` file has imports that start with `./` and `../`, then you don't require a module file. 

In theory, you can also use `./` and `../` imports with multiple repos in the local file system, but this can get ugly pretty fast.

All module files require publically accessible github.com repos at the moment. Eventually I'll add support for private repos, then other git repos, etc... Right now, I'm concentrating on getting this working and keeping that simple.

## General syntax example

```claw.mod
module github.com/repo/vehicles

claw 0.1

require (
	github.com/johnsiilver/trucks v1.1.1
	github.com/johnsiilver/cars v0.0.0-20220321173239-a90fa8a75705
	github.com/johnsiilver/motorcycles v1.1.0
)

replace (
    github.com/johnsiilver/motorcycles => ../../directory/of/module
)
```

This example has all the major components of the `claw.mod` file:

* `module <path>` is the name of this module. It must be the same as the `package` declaration in the `.claw` file.
* `claw <version>` is the version of the clawc compiler required to compile this .claw file.
* `require ()` provides a list of required modules. These should be the same as the import statements in the `.claw` file, except they contain version numbers afterwards. Imports that use `./` or `../` do not show up here. Neither do any other modules that are within the same repository. All `claw.mod` files in a single repository must use the same version of an imported module.
* `replace ()` provides the ability to replace a required module with the module on the local filesystem. This allows you to do local development on a dependent module while working on this module.
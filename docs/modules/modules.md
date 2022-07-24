# Claw Modules

## Introduction

Claw module files provide for versioning import dependencies imported in your `.claw` file and access control lists (ACLs) for what can use your `.claw` file.

`clawc` determines what version of the `.claw` file to use by first looking for the latest release version with highest number and if that doesn't exist, using the highest committed version in master/main. 

It is important to remember that `.claw` compilation is version independent from a project's version. For example, if you import a `.claw` file from project A that has current revision v3.4.2 to use against project A version v1.3.2, this should be fine. You will have later data models, but these should be backwards compatible due to the nature of an IDL. 

We thought about Claw having its own version numbers, separate from the repository, but that was just going to lead to a lot of commits that didn't update the version. When we introduce an RPC service layer, we will require a major revision number to target for service definitions.

An IDL such as Claw should never have changes made that remove anything. They should always be forward compatibile. If you need to create new cleaned up version, you really need a `v2/` folder.

`claw.mod` allows staticing your module file to specific versions of its dependencies to avoid these types of problems. 

The other major use case for the `claw.mod` is supporting ACLs. When I publish my `.claw` file, I may not want others to import it. It may be that I want it for my use only so that I can re-write the defintions in a non-backwards compatible way without worrying about breaking other users. 

By default, a `.claw` file is not importable by any other `.claw` file. To allow it to be imported, you must declare it to be publically accessible or list the packages that can import it. This prevents unintentional side effects. Note however that this does no prevent the language specific packages that are generated from being used.

`claw.mod` files are not required and only affect compilation from the directory specified. `claw.mod` files in other dependencies are ignored.

## General syntax example of claw.mod file

```claw.mod
require (
	github.com/johnsiilver/trucks v1.1.1
	github.com/johnsiilver/cars v0.2.5
	github.com/johnsiilver/motorcycles v1.1.0
)

replace (
	github.com/badactor/motorcycles => github.com/johnsiilver/motorcycles
)

acls (
	github.com/johnsiilver/vehicles/*
	github.com/djustice/vehicles/toyota
)
```

This example has all the major components of the `claw.mod` file:

* `require()` provides a list of required modules at some version. You only need to put in imports that need to be statically required.
* `replace()` provides for replacing a package that is imported with another package. This allows replacing a bad actor with a good copy. This differs from our local.replace file where
we replace locations for development and paths have to be local file paths. It is different that a global.replace, which specifies that the package location has moved. In our example, we replace one packag with another and specify the version for the replacemenbt in `require`.
* `acls()` provides a list of package paths that are allowed. This is either the fully qualified name or can end with a `/*` to note that any package underneath can import this.

The other option that can exist here is `acls = public` instead of `acls ()` which means anything can import this. As an owner, this mean this should:

* Never be removed at any version going forward to alway allow backwards compatibility. Remember, the user may have written records based on this to disk somewhere for long term storage long after your project is dead.
* Onlly backwards compatible changes, aka never remove anything, only add things.
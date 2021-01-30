Unikraft Tools
==============================

Unikraft is an automated system for building specialized OSes and
unikernels tailored to the needs of specific applications. It is based
around the concept of small, modular libraries, each providing a part
of the functionality commonly found in an operating system (e.g.,
memory allocation, scheduling, filesystem support, network stack,
etc.).

This repo contains all tools related to Unikraft, and in particular
the main.go which acts as a single point of entry for all Unikraft
operations, including the downloading, building and running
of Unikraft applications.

Note that this repo in general, is currently under heavy development
and should not yet be used unless you know what you are doing. As things 
stabilize, we will update this file to reflect this.

# Toolchain

Welcome to the Unikraft tools wiki!

The Unikraft tools are a set of tools to automatically build images of operating systems targeting applications. The toolchain will include the following tools:
1. **Decomposition tool** to assist developers in breaking existing monolithic software into smaller components.
2. **Dependency analysis tool** to analyse existing, unmodified applications to determine which set of libraries and OS primitives are absolutely necessary for correct execution.
3. **Automatic build tool** to match the requirements derived by the dependency analysis tools to the available libraries constructed by the OS decomposition tools. This one is composed of two components: a static analysis and a dynamic analysis.
4. **Verification tool** to ensure that the functionality of the resulting, specialized OS+application matches that of the application running on a standard OS. The tool will also take care of ensuring software quality.
5. **Performance optimization tool** to analyse the running specialized OS+application and to use this information as input to the automatic build tools so that they can generate even more optimized images.

In addition, the toolchain contains helper tools such as:
- **Crawler tool** to create graph of dependencies of existing micro-libs.
- **Binary analyser tool** to extract various information of unikraft unikernel ELF files.

Note that the toolchain will be integrated to the [kraft](https://github.com/unikraft/kraft) repository in February/March (after some refactoring).

## Installation and documentation

For installation and documentation, a wiki is available on this [address](https://github.com/gaulthiergain/tools/wiki).

## Contribute

Unikraft tools is an open source project (under MIT license) and is currently hosted at https://github.com/gaulthiergain/tools. You are encouraged to download the code, examine it, modify it, and submit bug reports, bug fixes, feature requests, new features and other issues and pull requests.
